package gridscale

import (
	"context"
	"fmt"

	"github.com/gridscale/gsclient-go"

	"github.com/hashicorp/packer/helper/multistep"
	"github.com/hashicorp/packer/packer"
)

type stepShutdown struct{}

func (s *stepShutdown) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	client := state.Get("client").(*gsclient.Client)
	//c := state.Get("config").(*Config)
	ui := state.Get("ui").(packer.Ui)
	serverID := state.Get("server_id").(string)

	ui.Say("Gracefully shutting down server...")
	err := client.ShutdownServer(context.Background(), serverID)
	if err != nil {
		// If we get an error the first time, actually report it
		err := fmt.Errorf("Error shutting down droplet: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	// err = waitForDropletState("off", dropletId, client, c.StateTimeout)
	// if err != nil {
	// 	// If we get an error the first time, actually report it
	// 	err := fmt.Errorf("Error shutting down droplet: %s", err)
	// 	state.Put("error", err)
	// 	ui.Error(err.Error())
	// 	return multistep.ActionHalt
	// }

	return multistep.ActionContinue
}

func (s *stepShutdown) Cleanup(state multistep.StateBag) {
	// no cleanup
}
