package main

// TODO: add a labels collector so we can see all labels across the swarm

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/client"
)

func main() {

	outputDir := "./out"

	log.Println("Starting Docker event listener")

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatalf("Error creating Docker client: %v", err)
	}

	ctx := context.Background()

	messages, errs := cli.Events(ctx, events.ListOptions{})

	go func() {
		for {
			select {
			case msg := <-messages:
				handleEvent(cli, msg)
			case err := <-errs:
				log.Printf("Error: %v", err)
				return
			}
		}
	}()

	select {} // block forever
}

func handleEvent(cli *client.Client, msg events.Message) {
	if msg.Action == "start" || msg.Action == "stop" || msg.Action == "update" {
		log.Printf("Received event: %s %s", msg.Action, msg.Actor.Attributes["name"])
		if msg.Type == events.ServiceEventType {
			serviceID := msg.Actor.ID
			service, raw, err := cli.ServiceInspectWithRaw(context.Background(), serviceID, types.ServiceInspectOptions{})
			if err != nil {
				log.Printf("Error inspecting service: %v", err)
				return
			}

			// Print the service configuration
			// log.Printf("Service configuration:\n%+v\n", service)

			// Optionally, print the raw JSON manifest
			// log.Println("Raw JSON manifest:")
			// log.Println(string(raw))

			// Dump the raw JSON to a file
			timestamp := time.Now().Format("20060102-150405")
			filePath := filepath.Join("./out", "service-"+serviceID+"-"+timestamp+".json")
			if err := os.WriteFile(filePath, raw, 0644); err != nil {
				log.Printf("Error writing raw JSON to file: %v", err)
				return
			}
			log.Printf("Raw JSON manifest written to: %s", filePath)

			// Generate Mermaid diagram
			mermaidDiagram := generateMermaidDiagram(service)
			diagramPath := filepath.Join("./out", "service-"+serviceID+"-"+timestamp+".mmd")
			if err := os.WriteFile(diagramPath, []byte(mermaidDiagram), 0644); err != nil {
				log.Printf("Error writing Mermaid diagram to file: %v", err)
				return
			}
			log.Printf("Mermaid diagram written to: %s", diagramPath)
		}
	}
}

func generateMermaidDiagram(service swarm.Service) string {
	diagram := "graph TD\n"
	diagram += fmt.Sprintf("Service[\"%s\"]\n", service.Spec.Name)

	// Add environment variables
	for _, env := range service.Spec.TaskTemplate.ContainerSpec.Env {
		diagram += fmt.Sprintf("Service -->|ENV| %s\n", env)
	}

	// Add volumes
	for _, mount := range service.Spec.TaskTemplate.ContainerSpec.Mounts {
		diagram += fmt.Sprintf("Service -->|VOLUME| %s\n", mount.Target)
	}

	// Add networks
	for _, network := range service.Spec.TaskTemplate.Networks {
		diagram += fmt.Sprintf("Service -->|NETWORK| %s\n", network.Target)
	}

	// Add ports
	for _, port := range service.Endpoint.Ports {
		diagram += fmt.Sprintf("Service -->|PORT| %d:%d\n", port.PublishedPort, port.TargetPort)
	}

	// Add labels
	for key, value := range service.Spec.Labels {
		diagram += fmt.Sprintf("Service -->|LABEL| %s: %s\n", key, value)
	}

	return diagram
}
