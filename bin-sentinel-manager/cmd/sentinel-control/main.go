package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	dockercontainer "github.com/docker/docker/api/types/container"
	dockerclient "github.com/docker/docker/client"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"monorepo/bin-sentinel-manager/internal/config"
)

func main() {
	cmd := initCommand()
	if errExecute := cmd.Execute(); errExecute != nil {
		log.Fatalf("Execution failed: %v", errExecute)
	}
}

func initCommand() *cobra.Command {
	cmdRoot := &cobra.Command{
		Use:   "sentinel-control",
		Short: "Voipbin Sentinel Management CLI",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if errBind := viper.BindPFlags(cmd.Flags()); errBind != nil {
				return errors.Wrap(errBind, "failed to bind flags")
			}

			config.LoadGlobalConfig()
			return nil
		},
	}

	if err := config.Bootstrap(cmdRoot); err != nil {
		cobra.CheckErr(errors.Wrap(err, "failed to bootstrap config"))
	}

	cmdContainer := &cobra.Command{Use: "container", Short: "Container monitoring operations"}
	cmdContainer.AddCommand(cmdContainerList())
	cmdContainer.AddCommand(cmdContainerGet())

	cmdRoot.AddCommand(cmdContainer)
	return cmdRoot
}

func cmdContainerList() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List containers visible through the docker-socket-proxy",
		RunE:  runContainerList,
	}

	flags := cmd.Flags()
	flags.String("name", "", "Substring filter on the container name (e.g., voip-asterisk-call-docker)")
	flags.Bool("all", false, "Include non-running containers")

	return cmd
}

func cmdContainerGet() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Inspect a specific container by name",
		RunE:  runContainerGet,
	}

	flags := cmd.Flags()
	flags.String("name", "", "Container name (required)")

	return cmd
}

// ContainerInfo is the summarized shape `container list` prints.
type ContainerInfo struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Image     string            `json:"image"`
	State     string            `json:"state"`
	Status    string            `json:"status"`
	Addresses map[string]string `json:"addresses"`
}

// ContainerDetailInfo is the detailed shape `container get` prints.
type ContainerDetailInfo struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Image     string            `json:"image"`
	State     string            `json:"state"`
	StartedAt string            `json:"started_at"`
	Addresses map[string]string `json:"addresses"`
	MACs      map[string]string `json:"mac_addresses"`
}

func runContainerList(cmd *cobra.Command, args []string) error {
	cli, err := newDockerClient()
	if err != nil {
		return err
	}
	defer closeDockerClient(cli)

	nameFilter := viper.GetString("name")
	all := viper.GetBool("all")

	containers, err := cli.ContainerList(context.Background(), dockercontainer.ListOptions{All: all})
	if err != nil {
		return errors.Wrap(err, "failed to list containers")
	}

	result := make([]ContainerInfo, 0, len(containers))
	for _, c := range containers {
		name := ""
		if len(c.Names) > 0 {
			name = strings.TrimPrefix(c.Names[0], "/")
		}

		if nameFilter != "" && !strings.Contains(name, nameFilter) {
			continue
		}

		addresses := map[string]string{}
		if c.NetworkSettings != nil {
			for network, endpoint := range c.NetworkSettings.Networks {
				if endpoint == nil {
					continue
				}
				addresses[network] = endpoint.IPAddress
			}
		}

		result = append(result, ContainerInfo{
			ID:        c.ID,
			Name:      name,
			Image:     c.Image,
			State:     string(c.State),
			Status:    c.Status,
			Addresses: addresses,
		})
	}

	return printJSON(result)
}

func runContainerGet(cmd *cobra.Command, args []string) error {
	name := viper.GetString("name")
	if name == "" {
		return fmt.Errorf("container name is required")
	}

	cli, err := newDockerClient()
	if err != nil {
		return err
	}
	defer closeDockerClient(cli)

	inspect, err := cli.ContainerInspect(context.Background(), name)
	if err != nil {
		return errors.Wrap(err, "failed to inspect container")
	}

	result := ContainerDetailInfo{
		Addresses: map[string]string{},
		MACs:      map[string]string{},
	}

	if inspect.ContainerJSONBase != nil {
		result.ID = inspect.ID
		result.Name = strings.TrimPrefix(inspect.Name, "/")
		if inspect.State != nil {
			result.State = inspect.State.Status
			result.StartedAt = inspect.State.StartedAt
		}
	}
	if inspect.Config != nil {
		result.Image = inspect.Config.Image
	}
	if inspect.NetworkSettings != nil {
		for network, endpoint := range inspect.NetworkSettings.Networks {
			if endpoint == nil {
				continue
			}
			result.Addresses[network] = endpoint.IPAddress
			result.MACs[network] = endpoint.MacAddress
		}
	}

	return printJSON(result)
}

// newDockerClient dials the read-only docker-socket-proxy, the same endpoint the service itself
// uses. This CLI never touches /var/run/docker.sock directly.
func newDockerClient() (*dockerclient.Client, error) {
	cli, err := dockerclient.NewClientWithOpts(
		dockerclient.WithHost(config.Get().DockerSocketProxyAddress),
		dockerclient.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create docker client")
	}

	return cli, nil
}

func closeDockerClient(cli *dockerclient.Client) {
	if errClose := cli.Close(); errClose != nil {
		log.Printf("failed to close docker client: %v", errClose)
	}
}

func printJSON(v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return errors.Wrap(err, "failed to marshal JSON")
	}
	fmt.Println(string(data))
	return nil
}
