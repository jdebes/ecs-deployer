package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecs"
)

type arrayFlag []string

func (flags *arrayFlag) String() string {
	return strings.Join(*flags, ",")
}

func (flags *arrayFlag) Set(value string) error {
	*flags = append(*flags, value)
	return nil
}

func (flags *arrayFlag) Specified() bool {
	return len(*flags) > 0
}

var (
	clusterName  = flag.String("c", "", "Cluster name to deploy to")
	repoName     = flag.String("i", "", "Docker repository to pull from")
	sha          = flag.String("s", "latest", "Tag, usually short git SHA to deploy")
	region       = flag.String("r", "", "AWS region")
	debug        = flag.Bool("d", false, "enable Debug output")
)

var apps arrayFlag

func fail(s string) {
	fmt.Printf(s)
	os.Exit(2)
}

func init() {
	flag.Var(&apps, "a", "Application names (can be specified multiple times)")
}

func main() {
	flag.Parse()

	if *clusterName == "" || !apps.Specified() || *region == "" {
		flag.Usage()
		fail(fmt.Sprintf("Failed deployment of apps %s : missing parameters\n", apps))
	}

	if *repoName == "" || *sha == "" {
		flag.Usage()
		fail(fmt.Sprintf("Failed deployment %s : no repo name, sha or target image specified\n", apps))
	}

	// Take the first app specified and use it for creating the task definitions for all services.
	exemplarServiceName := apps[0]
	cfg := &aws.Config{
		Region: aws.String(*region),
	}
	if *debug {
		cfg = cfg.WithLogLevel(aws.LogDebug)
	}

	svc := ecs.New(session.New(), cfg)

	fmt.Printf("Request to deploy sha: %s at %s \n", *sha, *region)
	fmt.Printf("Describing services for cluster %s and service %s \n", *clusterName, exemplarServiceName)

	serviceDesc, err :=
		svc.DescribeServices(
			&ecs.DescribeServicesInput{
				Cluster:  clusterName,
				Services: []*string{&exemplarServiceName},
			})
	if err != nil {
		fail(fmt.Sprintf("Failed to describe %s \n`%s`", exemplarServiceName, err.Error()))
	}

	if len(serviceDesc.Services) < 1 {
		msg := fmt.Sprintf("No service %s found on cluster %s", exemplarServiceName, *clusterName)
		fail("Failed: " + msg)
	}

	service := serviceDesc.Services[0]
	if exemplarServiceName != *service.ServiceName {
		msg := fmt.Sprintf("Found the wrong service when looking for %s found %s \n", exemplarServiceName, *service.ServiceName)
		fail("Failed: " + msg)
	}

	fmt.Printf("Found existing ARN %s for service %s \n", *service.ClusterArn, *service.ServiceName)

	taskDesc, err :=
		svc.DescribeTaskDefinition(
			&ecs.DescribeTaskDefinitionInput{
				TaskDefinition: service.TaskDefinition})
	if err != nil {
		fail(fmt.Sprintf("Failed: deployment %s \n`%s`", exemplarServiceName, err.Error()))
	}

	if *debug {
		fmt.Printf("Current task description: \n%+v \n", taskDesc)
	}

	imageName := fmt.Sprintf("%s:%s", *repoName, *sha)
	containerDef := taskDesc.TaskDefinition.ContainerDefinitions[0]
	containerDef.Image = &imageName

	futureDef := &ecs.RegisterTaskDefinitionInput{
		ContainerDefinitions:    taskDesc.TaskDefinition.ContainerDefinitions,
		Family:                  taskDesc.TaskDefinition.Family,
		Volumes:                 taskDesc.TaskDefinition.Volumes,
		NetworkMode:             taskDesc.TaskDefinition.NetworkMode,
		TaskRoleArn:             taskDesc.TaskDefinition.TaskRoleArn,
		Cpu:                     taskDesc.TaskDefinition.Cpu,
		Memory:                  taskDesc.TaskDefinition.Memory,
		RequiresCompatibilities: taskDesc.TaskDefinition.RequiresCompatibilities,
		ExecutionRoleArn:        taskDesc.TaskDefinition.ExecutionRoleArn,
	}

	if *debug {
		fmt.Printf("Future task description: \n%+v \n", futureDef)
	}

	registerRes, err := svc.RegisterTaskDefinition(futureDef)
	if err != nil {
		fail(fmt.Sprintf("Failed: deployment %s for %s to %s \n`%s`", *containerDef.Image, exemplarServiceName, *clusterName, err.Error()))
	}

	newArn := registerRes.TaskDefinition.TaskDefinitionArn

	fmt.Printf("Registered new task for %s:%s \n", *sha, *newArn)

	// update services to use new definition
	for _, appName := range apps {
		serviceName := appName

		_, err = svc.UpdateService(
			&ecs.UpdateServiceInput{
				Cluster:        clusterName,
				Service:        &serviceName,
				DesiredCount:   service.DesiredCount,
				TaskDefinition: newArn,
			})
		if err != nil {
			fail(fmt.Sprintf("Failed: deployment %s for %s to %s as %s \n`%s`", *containerDef.Image, appName, *clusterName, *newArn, err.Error()))
		}

		fmt.Printf("Updated %s service to use new ARN: %s \n", serviceName, *newArn)
	}

}
