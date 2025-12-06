package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	enginev1 "juki-engine/gen/engine/v1"
	"juki-engine/gen/engine/v1/enginev1connect"

	"connectrpc.com/connect"
)

func main() {
	client := enginev1connect.NewEngineServiceClient(
		http.DefaultClient,
		"http://localhost:8080",
	)

	// cwd, _ := os.Getwd()
	// We want to create "test-project" in the repo root.
	// Engine is in .juki/engine. So repo root is ../../
	// But the engine resolves relative paths from its CWD.
	// Let's try passing an absolute path or a relative one.
	// The engine code uses the path as is.

	// Let's use a relative path that points to repo root
	targetPath := "../../test-project"

	fmt.Printf("Requesting project creation at: %s\n", targetPath)

	req := connect.NewRequest(&enginev1.CreateProjectRequest{
		Name:      "test-project",
		Path:      targetPath,
		Framework: "nextjs",
	})

	res, err := client.CreateProject(context.Background(), req)
	if err != nil {
		log.Fatalf("Failed to create project: %v", err)
	}

	fmt.Printf("Project created! ID: %s, Path: %s\n", res.Msg.Project.Id, res.Msg.Project.Path)

	// Verify folder exists
	// We need to resolve the path relative to where WE are running (which is .juki/engine)
	absPath, _ := filepath.Abs(targetPath)
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		log.Fatalf("Error: Project directory %s does not exist!", absPath)
	}

	fmt.Println("Verification Successful: Project directory exists.")
}
