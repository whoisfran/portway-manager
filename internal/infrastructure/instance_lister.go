package infrastructure

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ssm"

	"ssm-portway/internal/domain"
	"ssm-portway/models"
)

// awsInstanceLister consulta SSM DescribeInstanceInformation y lo
// cruza con EC2 DescribeInstances para enriquecer cada instancia con
// su tag "Name".
type awsInstanceLister struct{}

func NewAWSInstanceLister() domain.InstanceLister {
	return &awsInstanceLister{}
}

func (l *awsInstanceLister) List(ctx context.Context, profile, region string) ([]models.Instance, error) {
	var opts []func(*config.LoadOptions) error
	if profile != "" {
		opts = append(opts, config.WithSharedConfigProfile(profile))
	}
	if region != "" {
		opts = append(opts, config.WithRegion(region))
	}

	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("no se pudo cargar la configuracion de AWS: %w", err)
	}

	ssmClient := ssm.NewFromConfig(cfg)
	ec2Client := ec2.NewFromConfig(cfg)

	nameByID := map[string]string{}

	ec2Paginator := ec2.NewDescribeInstancesPaginator(ec2Client, &ec2.DescribeInstancesInput{})
	for ec2Paginator.HasMorePages() {
		page, err := ec2Paginator.NextPage(ctx)
		if err != nil {
			break
		}
		for _, res := range page.Reservations {
			for _, inst := range res.Instances {
				name := ""
				for _, tag := range inst.Tags {
					if tag.Key != nil && *tag.Key == "Name" && tag.Value != nil {
						name = *tag.Value
					}
				}
				if inst.InstanceId != nil {
					nameByID[*inst.InstanceId] = name
				}
			}
		}
	}

	var instances []models.Instance
	ssmPaginator := ssm.NewDescribeInstanceInformationPaginator(ssmClient, &ssm.DescribeInstanceInformationInput{})
	for ssmPaginator.HasMorePages() {
		page, err := ssmPaginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("no se pudo listar instancias SSM: %w", err)
		}
		for _, info := range page.InstanceInformationList {
			id := ""
			if info.InstanceId != nil {
				id = *info.InstanceId
			}
			ip := ""
			if info.IPAddress != nil {
				ip = *info.IPAddress
			}
			instances = append(instances, models.Instance{
				InstanceID: id,
				Name:       nameByID[id],
				PlatformOS: string(info.PlatformType),
				PrivateIP:  ip,
				PingStatus: string(info.PingStatus),
			})
		}
	}

	return instances, nil
}
