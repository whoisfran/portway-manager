package ssm

import (
	"encoding/json"
	"reflect"
	"testing"

	"portway-manager/models"
)

// ssmSampleJSON y ec2SampleJSON son recortes reales de "aws ssm
// describe-instance-information --output json" / "aws ec2
// describe-instances --output json" (solo los campos que a esta app le
// importan), para confirmar que las structs decodifican el JSON real
// del CLI y no solo datos armados a mano.
const ssmSampleJSON = `{
	"InstanceInformationList": [
		{"InstanceId": "i-aaa", "PingStatus": "Online", "PlatformType": "Windows", "IPAddress": "10.0.0.1"},
		{"InstanceId": "i-bbb", "PingStatus": "ConnectionLost", "PlatformType": "Linux", "IPAddress": "10.0.0.2"}
	]
}`

const ec2SampleJSON = `{
	"Reservations": [
		{"Instances": [
			{"InstanceId": "i-aaa", "Tags": [{"Key": "Environment", "Value": "prod"}, {"Key": "Name", "Value": "db-01"}]}
		]}
	]
}`

func TestBuildInstancesCrossReferencesNameByTag(t *testing.T) {
	var ssmOut ssmDescribeInstanceInformationOutput
	if err := json.Unmarshal([]byte(ssmSampleJSON), &ssmOut); err != nil {
		t.Fatalf("no se pudo decodificar el JSON de SSM: %v", err)
	}
	var ec2Out ec2DescribeInstancesOutput
	if err := json.Unmarshal([]byte(ec2SampleJSON), &ec2Out); err != nil {
		t.Fatalf("no se pudo decodificar el JSON de EC2: %v", err)
	}

	got := buildInstances(ssmOut, ec2Out)
	want := []models.Instance{
		{InstanceID: "i-aaa", Name: "db-01", PlatformOS: "Windows", PrivateIP: "10.0.0.1", PingStatus: "Online"},
		// i-bbb no aparece en el resultado de EC2 (nunca se llego a
		// consultar, o no tiene tag "Name"): se queda sin nombre, pero
		// igual aparece en la lista -- SSM es la fuente de verdad de
		// que instancias mostrar.
		{InstanceID: "i-bbb", Name: "", PlatformOS: "Linux", PrivateIP: "10.0.0.2", PingStatus: "ConnectionLost"},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("buildInstances() = %+v, want %+v", got, want)
	}
}

// TestBuildInstancesWithoutEC2Data cubre el caso real de este
// proyecto: falta el permiso ec2:DescribeInstances (List() ignora ese
// error, ver comentario en List), asi que ec2Out llega vacio. Las
// instancias deben seguir apareciendo, solo sin nombre.
func TestBuildInstancesWithoutEC2Data(t *testing.T) {
	var ssmOut ssmDescribeInstanceInformationOutput
	if err := json.Unmarshal([]byte(ssmSampleJSON), &ssmOut); err != nil {
		t.Fatalf("no se pudo decodificar el JSON de SSM: %v", err)
	}

	got := buildInstances(ssmOut, ec2DescribeInstancesOutput{})

	if len(got) != 2 {
		t.Fatalf("len(got) = %d, want 2", len(got))
	}
	for _, inst := range got {
		if inst.Name != "" {
			t.Errorf("instancia %s: Name = %q, want vacio (sin datos de EC2)", inst.InstanceID, inst.Name)
		}
	}
}
