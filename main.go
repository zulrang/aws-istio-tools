package main

import (
	"log"
)

func main() {

	aws, err := NewAWSClient()
	if err != nil {
		log.Fatal(err)
	}

	if err := ReconcileIstioServicePorts(aws, "tkg-dev-shared", "istio-system/istio-eastwestgateway"); err != nil {
		log.Fatalf("unable to reconcile istio service ports, %v", err)
	}

}
