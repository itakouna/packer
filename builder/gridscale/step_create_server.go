package gridscale

import (
	"context"
	"fmt"

	"github.com/gridscale/gsclient-go"

	"github.com/hashicorp/packer/helper/multistep"
	"github.com/hashicorp/packer/packer"
)

type stepCreateServer struct {
	serverId  string
	storageId string
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
		Cores:  c.ServerCores,
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
	ui.Say("Creating Storage...")
	//Create a storage
	template := gsclient.StorageTemplate{
		Password:     c.Password,
		PasswordType: gsclient.PlainPasswordType,
		Hostname:     c.Hostname,
		TemplateUUID: c.TemplateUUID,
	}

	storage, err := client.CreateStorage(
		context.Background(),
		gsclient.StorageCreateRequest{
			Capacity:    c.StorageCapacity,
			Name:        c.ServerName,
			StorageType: gsclient.InsaneStorageType,
			Template:    &template,
		})
	if err != nil {
		ui.Error(fmt.Sprintf(
			"Error creating storage: %s", err))
	}
	s.storageId = storage.ObjectUUID
	state.Put("storage_id", storage.ObjectUUID)
	ui.Say("Link Server with Storage...")

	client.LinkStorage(context.Background(), server.ObjectUUID, storage.ObjectUUID, true)
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

	// Destroy the storage we just created
	ui.Say("Destroying storage...")
	err = client.DeleteStorage(context.TODO(), s.storageId)
	if err != nil {
		ui.Error(fmt.Sprintf(
			"Error destroying storage. Please destroy it manually: %s", err))
	}
}
