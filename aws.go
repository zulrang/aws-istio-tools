package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi"
	"github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi/types"
)

type AWSClient struct {
	elb *elasticloadbalancingv2.Client
	ec2 *ec2.Client
	tg  *resourcegroupstaggingapi.Client
	cfg *aws.Config
}

func NewAWSClient() (*AWSClient, error) {

	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return nil, fmt.Errorf("unable to load SDK config, %v", err)
	}

	return &AWSClient{cfg: &cfg}, nil
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

func (aws *AWSClient) GetTaggedNodeInstanceIds(cluster string) ([]string, error) {
	if aws.ec2 == nil {
		aws.ec2 = ec2.NewFromConfig(*aws.cfg)
	}

	result, err := aws.ec2.DescribeInstances(context.TODO(), &ec2.DescribeInstancesInput{
		Filters: mapToFilter(getNodeMapTags(cluster)),
	})

	if err != nil {
		return nil, fmt.Errorf("unable to get instances, %v", err)
	}

	instance_ids := []string{}
	for _, r := range result.Reservations {
		for _, i := range r.Instances {
			instance_ids = append(instance_ids, *i.InstanceId)
		}
	}

	return instance_ids, nil
}

func mapToTagFilter(m map[string][]string) []types.TagFilter {
	tagFilters := []types.TagFilter{}
	for k, v := range m {
		key := k
		tagFilters = append(tagFilters, types.TagFilter{Key: &key, Values: v})
	}
	return tagFilters
}

func mapToFilter(m map[string][]string) []ec2types.Filter {
	filters := []ec2types.Filter{}
	for k, v := range m {
		key := fmt.Sprintf("tag:%v", k)
		filters = append(filters, ec2types.Filter{Name: &key, Values: v})
	}
	return filters
}

func getLBMapTags(cluster string, service string) map[string][]string {
	return map[string][]string{
		"kubernetes.io/service-name":                     {service},
		fmt.Sprintf("kubernetes.io/cluster/%v", cluster): {"owned"},
	}
}

func getNodeMapTags(cluster string) map[string][]string {
	return map[string][]string{
		"sigs.k8s.io/cluster-api-provider-aws/role":      {"node"},
		fmt.Sprintf("kubernetes.io/cluster/%v", cluster): {"owned"},
	}
}

func (aws *AWSClient) GetNLBARNs(clusterName string, serviceName string) ([]string, error) {
	if aws.tg == nil {
		aws.tg = resourcegroupstaggingapi.NewFromConfig(*aws.cfg)
	}

	result, err := aws.tg.GetResources(context.TODO(), &resourcegroupstaggingapi.GetResourcesInput{
		TagFilters: mapToTagFilter(getLBMapTags(clusterName, serviceName)),
	})

	if err != nil {
		return nil, fmt.Errorf("unable to get resources, %v", err)
	}

	nlb_arns := []string{}
	for _, rtm := range result.ResourceTagMappingList {
		// only return NLBs
		if strings.Contains(*rtm.ResourceARN, "loadbalancer/net") {
			nlb_arns = append(nlb_arns, *rtm.ResourceARN)
		}
	}

	return nlb_arns, nil
}
