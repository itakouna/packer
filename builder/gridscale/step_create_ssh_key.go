package gridscale

import (
	"context"
	"fmt"
	"log"

	"github.com/gridscale/gsclient-go"

	"github.com/hashicorp/packer/common/uuid"
	"github.com/hashicorp/packer/helper/multistep"
	"github.com/hashicorp/packer/packer"
)

type stepCreateSSHKey struct {
	Debug        bool
	DebugKeyPath string
	keyId        string
}

func (s *stepCreateSSHKey) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	client := state.Get("client").(*gsclient.Client)
	ui := state.Get("ui").(packer.Ui)
	c := state.Get("config").(*Config)

	ui.Say("Creating temporary ssh key for server...")
	ui.Say(c.SSHKey)
	// Create the key!
	resp, err := client.CreateSshkey(context.TODO(), gsclient.SshkeyCreateRequest{
		Name:   fmt.Sprintf("packer-%s", uuid.TimeOrderedUUID()),
		Sshkey: c.SSHKey,
	})
	if err != nil {
		err := fmt.Errorf("Error getting temporary SSH key: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	// We use this to check cleanup
	s.keyId = resp.ObjectUUID

	log.Printf("temporary ssh key name: %s", resp.ObjectUUID)

	// Remember some state for the future
	state.Put("ssh_key_id", resp.ObjectUUID)

	return multistep.ActionContinue
}

func (s *stepCreateSSHKey) Cleanup(state multistep.StateBag) {
	// If no key name is set, then we never created it, so just return
	if s.keyId == "" {
		return
	}

	client := state.Get("client").(*gsclient.Client)
	ui := state.Get("ui").(packer.Ui)

	ui.Say("Deleting temporary ssh key...")
	err := client.DeleteSshkey(context.TODO(), s.keyId)
	if err != nil {
		log.Printf("Error cleaning up ssh key: %s", err)
		ui.Error(fmt.Sprintf(
			"Error cleaning up ssh key. Please delete the key manually: %s", err))
	}
}
