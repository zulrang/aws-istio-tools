package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi"
	"github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi/types"
)

type AWSClient struct {
	elb *elasticloadbalancingv2.Client
	tg  *resourcegroupstaggingapi.Client
	cfg *aws.Config
}

func (aws *AWSClient) GetTargetGroupARN(loadBalancerARN string) (string, error) {
	if aws.elb == nil {
		aws.elb = elasticloadbalancingv2.NewFromConfig(*aws.cfg)
	}

	lb_result, err := aws.elb.DescribeLoadBalancers(context.TODO(), &elasticloadbalancingv2.DescribeLoadBalancersInput{
		LoadBalancerArns: []string{loadBalancerARN},
	})

	if err != nil {
		return "", err
	}

	if len(lb_result.LoadBalancers) != 1 {
		return "", fmt.Errorf("unable to find NLB")
	}

	lb := lb_result.LoadBalancers[0]
	tg_result, err := aws.elb.DescribeTargetGroups(context.TODO(), &elasticloadbalancingv2.DescribeTargetGroupsInput{
		LoadBalancerArn: lb.LoadBalancerArn,
	})

	if err != nil {
		return "", err
	}

	tgs := tg_result.TargetGroups
	if len(tgs) != 1 {
		return "", fmt.Errorf("unable to find target group")
	}

	tg := tgs[0]

	return *tg.TargetGroupArn, nil
}

func (aws *AWSClient) GetNLBARNs(clusterName string, serviceName string) ([]string, error) {
	if aws.tg == nil {
		aws.tg = resourcegroupstaggingapi.NewFromConfig(*aws.cfg)
	}

	tags_to_find := map[string][]string{
		"kubernetes.io/service-name":                         []string{serviceName},
		fmt.Sprintf("kubernetes.io/cluster/%v", clusterName): []string{"owned"},
	}

	tagFilters := []types.TagFilter{}
	for k, v := range tags_to_find {
		key := k
		tagFilters = append(tagFilters, types.TagFilter{Key: &key, Values: v})
	}

	result, err := aws.tg.GetResources(context.TODO(), &resourcegroupstaggingapi.GetResourcesInput{
		TagFilters: tagFilters,
	})

	if err != nil {
		return nil, err
	}

	nlb_arns := []string{}
	for _, rtm := range result.ResourceTagMappingList {
		if strings.Contains(*rtm.ResourceARN, "loadbalancer/net") {
			nlb_arns = append(nlb_arns, *rtm.ResourceARN)
		}
	}

	return nlb_arns, nil
}

func ReconcileIstioServicePorts(aws *AWSClient) error {

	nlb_arns, err := aws.GetNLBARNs("tkg-dev-shared", "istio-system/istio-eastwestgateway")

	if err != nil {
		return fmt.Errorf("unable to get NLB ARNs, %v", err)
	}

	if len(nlb_arns) == 0 {
		// TODO: create NLB
		return fmt.Errorf("unable to find NLB ARN")
	}

	if len(nlb_arns) > 1 {
		return fmt.Errorf("found more than one NLB ARN")
	}

	nlb_arn := nlb_arns[0]

	fmt.Println(nlb_arn)

	tg_arn, err := aws.GetTargetGroupARN(nlb_arn)

	if err != nil {
		return fmt.Errorf("unable to get target group ARN, %v", err)
	}

	fmt.Println(tg_arn)

	return nil
}

func main() {
	// Using the SDK's default configuration, loading additional config
	// and credentials values from the environment variables, shared
	// credentials, and shared configuration files
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}

	aws := AWSClient{cfg: &cfg}

	if err := ReconcileIstioServicePorts(&aws); err != nil {
		log.Fatalf("unable to reconcile istio service ports, %v", err)
	}

}
