package main

import "fmt"

func ReconcileIstioServicePorts(aws *AWSClient, cluster string, service string) error {

	// get existing NLBs
	// create target group if it doesn't exist
	// register instances to target group
	// register target group to NLB

	nlb_arns, err := aws.GetNLBARNs(cluster, service)

	if err != nil {
		return fmt.Errorf("unable to get NLB ARNs, %v", err)
	}

	if len(nlb_arns) > 1 {
		return fmt.Errorf("found more than one NLB ARN")
	}

	// create NLB if it doesn't exist
	if len(nlb_arns) == 0 {
		// TODO: create NLB
		return fmt.Errorf("unable to find NLB ARN")
	}

	nlb_arn := nlb_arns[0]

	// get existing target groups
	tg_arn, err := aws.GetTargetGroupARN(nlb_arn)

	if err != nil {
		return fmt.Errorf("unable to get target group ARN, %v", err)
	}

	fmt.Println(tg_arn)

	return nil
}
