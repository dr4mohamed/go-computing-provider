package computing

import (
	"context"
	"github.com/compose-spec/compose-go/v2/loader"
	comp_type "github.com/compose-spec/compose-go/v2/types"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/flags"
	"github.com/docker/compose/v2/pkg/api"
	"github.com/docker/compose/v2/pkg/compose"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type ComposeApiService struct {
	api.Service
}

func NewComposeService() (*ComposeApiService, error) {
	var srv api.Service
	dockerCli, err := command.NewDockerCli()
	if err != nil {
		return nil, err
	}

	//Magic line to fix error:
	//Failed to initialize: unable to resolve docker endpoint: no context store initialized
	myOpts := &flags.ClientOptions{Context: "default", LogLevel: "error"}
	err = dockerCli.Initialize(myOpts)
	if err != nil {
		return nil, err
	}

	srv = compose.NewComposeService(dockerCli)
	return &ComposeApiService{srv}, err
}

func (cas *ComposeApiService) ServiceUp(dockerComposeBody string) error {
	project, err := loader.LoadWithContext(context.TODO(), comp_type.ConfigDetails{
		ConfigFiles: []comp_type.ConfigFile{
			{Filename: "docker-compose.yaml", Content: []byte(dockerComposeBody)},
		},
	}, func(options *loader.Options) {
		options.SetProjectName("install-pre-dependency-env", true)
	})

	if err != nil {
		return err
	}
	return cas.Up(context.TODO(), project, api.UpOptions{
		//Start: api.StartOptions{
		//	Services: []string{"redis", "resource-exporter"},
		//	Wait:     true,
		//	Watch:    true,
		//},
	})
}

func (cas *ComposeApiService) ServiceDown(dockerComposeBody string) error {
	project, err := loader.LoadWithContext(context.TODO(), comp_type.ConfigDetails{
		ConfigFiles: []comp_type.ConfigFile{
			{Filename: "docker-compose.yaml", Content: []byte(dockerComposeBody)},
		},
	}, func(options *loader.Options) {
		options.SetProjectName("stop-pre-dependency-env", true)
	})

	if err != nil {
		return err
	}
	return cas.Down(context.TODO(), "stop-pre-dependency-env", api.DownOptions{
		RemoveOrphans: true,
		Project:       project,
	})
}

func StopPreviousServices(dockerComposeContent, cpPath string) error {
	err := extractComposeFile(dockerComposeContent, cpPath)
	if err != nil {
		return err
	}
	cmd := exec.Command("docker-compose", "-f", filepath.Join(cpPath, "docker-compose.yaml"), "down")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}

func RunDockerCompose(dockerComposeContent, cpPath string) error {
	err := extractComposeFile(dockerComposeContent, cpPath)
	if err != nil {
		return err
	}

	cmd := exec.Command("docker-compose", "-f", filepath.Join(cpPath, "docker-compose.yaml"), "up", "-d")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

func extractComposeFile(dockerComposeContent, cpPath string) error {
	filePath := filepath.Join(cpPath, "docker-compose.yaml")
	if _, err := os.Stat(filePath); err != nil {
		dockerComposeContent = strings.ReplaceAll(dockerComposeContent, "$CP_PATH", cpPath)
		if err = os.WriteFile(filePath, []byte(dockerComposeContent), 0644); err != nil {
			return err
		}
	}
	return nil
}
