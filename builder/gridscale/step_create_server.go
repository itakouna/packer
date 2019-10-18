package gridscale

import (
	"context"
	"fmt"

	"github.com/gridscale/gsclient-go"

	"github.com/hashicorp/packer/helper/multistep"
	"github.com/hashicorp/packer/packer"
)

type stepCreateServer struct {
	serverId string
}

func (s *stepCreateServer) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	client := state.Get("client").(*gsclient.Client)
	ui := state.Get("ui").(packer.Ui)
	c := state.Get("config").(*Config)
	//sshKeyId := state.Get("ssh_key_id").(int)

	// Create the server based on configuration
	ui.Say("Creating server...")

	server, err := client.CreateServer(context.Background(), gsclient.ServerCreateRequest{
		Name:   c.ServerName,
		Cores: c.ServerCores,
		Memory: c.ServerMemory,
	})
	if err != nil {
		err := fmt.Errorf("Error creating server: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	// We use this in cleanup
	s.serverId = server.ObjectUUID

	// Store the server id for later
	state.Put("server_id", server.ObjectUUID)

	return multistep.ActionContinue
}

func (s *stepCreateServer) Cleanup(state multistep.StateBag) {
	// If the serverid isn't there, we probably never created it
	if s.serverId == "" {
		return
	}

	client := state.Get("client").(*gsclient.Client)
	ui := state.Get("ui").(packer.Ui)

	// Destroy the server we just created
	ui.Say("Destroying server...")
	err := client.DeleteServer(context.TODO(), s.serverId)
	if err != nil {
		ui.Error(fmt.Sprintf(
			"Error destroying server. Please destroy it manually: %s", err))
	}
}
