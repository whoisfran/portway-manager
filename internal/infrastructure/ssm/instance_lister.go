package ssm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"portway-manager/internal/domain"
	"portway-manager/models"
)

// awsInstanceLister consulta "aws ssm describe-instance-information"
// para las instancias administradas y las cruza con "aws ec2
// describe-instances" para enriquecerlas con su tag "Name".
//
// Usa el mismo CLI que la app ya exige como prerequisito para abrir
// tuneles SSM (ver prerequisites_checker.go y strategy.go), en vez del
// SDK de Go: el SDK de AWS es el unico motivo por el que este binario
// cargaba ~20MB extra (aws-sdk-go-v2 + toda su cadena de credenciales)
// para dos llamadas de solo lectura que el CLI ya hace igual de bien.
// El costo es que cada llamada tarda mas (el CLI en Python arranca en
// frio cada vez), pero esto no se dispara en un loop, solo cuando el
// usuario abre el selector de instancias.
type awsInstanceLister struct{}

func NewAWSInstanceLister() domain.InstanceLister {
	return &awsInstanceLister{}
}

// Las siguientes dos structs solo mapean los campos de la respuesta
// JSON del CLI que esta app usa; el resto de los campos que devuelve
// AWS se ignoran (encoding/json no exige que esten declarados todos).
type ssmDescribeInstanceInformationOutput struct {
	InstanceInformationList []struct {
		InstanceID   string `json:"InstanceId"`
		PingStatus   string `json:"PingStatus"`
		PlatformType string `json:"PlatformType"`
		IPAddress    string `json:"IPAddress"`
	} `json:"InstanceInformationList"`
}

type ec2DescribeInstancesOutput struct {
	Reservations []struct {
		Instances []struct {
			InstanceID string `json:"InstanceId"`
			Tags       []struct {
				Key   string `json:"Key"`
				Value string `json:"Value"`
			} `json:"Tags"`
		} `json:"Instances"`
	} `json:"Reservations"`
}

func (l *awsInstanceLister) List(ctx context.Context, profile, region string) ([]models.Instance, error) {
	// SSM no conoce los tags de EC2 (no sabe que una instancia se
	// llama "prod-db-01", solo su ID): se cruza por separado. Un
	// fallo aqui (p.ej. falta el permiso ec2:DescribeInstances, un
	// permiso totalmente separado de los de SSM) no debe impedir
	// listar las instancias -- solo se pierde el nombre bonito.
	var ec2Out ec2DescribeInstancesOutput
	_ = runAWSCLIJSON(ctx, &ec2Out, profile, region, "ec2", "describe-instances")

	var ssmOut ssmDescribeInstanceInformationOutput
	if err := runAWSCLIJSON(ctx, &ssmOut, profile, region, "ssm", "describe-instance-information"); err != nil {
		return nil, fmt.Errorf("no se pudo listar instancias SSM: %w", err)
	}

	return buildInstances(ssmOut, ec2Out), nil
}

// buildInstances cruza el resultado de las dos consultas: la lista de
// instancias administradas viene de SSM, el nombre "bonito" (tag
// "Name") de EC2 -- ec2Out puede venir vacio (ver List) sin que eso
// impida construir la lista, solo faltaran los nombres.
func buildInstances(ssmOut ssmDescribeInstanceInformationOutput, ec2Out ec2DescribeInstancesOutput) []models.Instance {
	nameByID := map[string]string{}
	for _, res := range ec2Out.Reservations {
		for _, inst := range res.Instances {
			for _, tag := range inst.Tags {
				if tag.Key == "Name" {
					nameByID[inst.InstanceID] = tag.Value
				}
			}
		}
	}

	instances := make([]models.Instance, 0, len(ssmOut.InstanceInformationList))
	for _, info := range ssmOut.InstanceInformationList {
		instances = append(instances, models.Instance{
			InstanceID: info.InstanceID,
			Name:       nameByID[info.InstanceID],
			PlatformOS: info.PlatformType,
			PrivateIP:  info.IPAddress,
			PingStatus: info.PingStatus,
		})
	}

	return instances
}

// runAWSCLIJSON corre "aws <service> <operation> [--profile ...]
// [--region ...] --output json" y decodifica su salida en out. El CLI
// pagina solo por defecto (a diferencia del SDK, no hace falta
// repetir la llamada pagina por pagina: ya devuelve todo combinado).
func runAWSCLIJSON(ctx context.Context, out any, profile, region, service, operation string) error {
	args := []string{service, operation, "--output", "json"}
	if profile != "" {
		args = append(args, "--profile", profile)
	}
	if region != "" {
		args = append(args, "--region", region)
	}

	cmd := exec.CommandContext(ctx, "aws", args...)
	stdout, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && len(exitErr.Stderr) > 0 {
			return fmt.Errorf("%s", strings.TrimSpace(string(exitErr.Stderr)))
		}
		return err
	}

	return json.Unmarshal(stdout, out)
}
