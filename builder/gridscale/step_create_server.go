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
	ipID      string
}

func (s *stepCreateServer) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	client := state.Get("client").(*gsclient.Client)
	ui := state.Get("ui").(packer.Ui)
	c := state.Get("config").(*Config)
	sshKeyId := state.Get("ssh_key_id").(string)

	publicNetwork, err := client.GetNetworkPublic(context.Background())
	if err != nil {
		ui.Error(fmt.Sprintf(
			"Error getting publicNetwork: %s", err))
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	// Create the server based on configuration
	ui.Say("Creating server...")
	requestBody := gsclient.ServerCreateRequest{
		Name:   c.ServerName,
		Cores:  c.ServerCores,
		Memory: c.ServerMemory,
	}

	server, err := client.CreateServer(context.Background(), requestBody)
	if err != nil {
		err := fmt.Errorf("Error creating server: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	s.serverId = server.ObjectUUID
	state.Put("server_id", server.ObjectUUID)

	ui.Say("Creating Storage...")
	//Create a storage
	var sshkeys []string
	template := gsclient.StorageTemplate{
		Password:     c.Password,
		PasswordType: gsclient.PlainPasswordType,
		Hostname:     c.Hostname,
		Sshkeys:      append(sshkeys, sshKeyId),
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
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	s.storageId = storage.ObjectUUID
	state.Put("storage_id", storage.ObjectUUID)

	ipRequest := gsclient.IPCreateRequest{
		Name:   c.ServerName,
		Family: gsclient.IPv4Type,
	}
	//Create new IP
	ui.Say("Create IP...")
	ip, err := client.CreateIP(context.Background(), ipRequest)

	if err != nil {
		ui.Error(fmt.Sprintf(
			"Error creating storage: %s", err))
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	s.ipID = ip.ObjectUUID
	state.Put("server_ip", ip.IP)

	ui.Say("Link Server with IP...")
	client.LinkIP(context.Background(), server.ObjectUUID, ip.ObjectUUID)

	ui.Say("Link Server with Storage...")
	client.LinkStorage(context.Background(), server.ObjectUUID, storage.ObjectUUID, true)

	ui.Say("Link Server with publicNetwork...")
	client.LinkNetwork(context.Background(), server.ObjectUUID, publicNetwork.Properties.ObjectUUID, "", false, 0, nil, nil)

	ui.Say("Start Server...")
	client.StartServer(context.Background(), server.ObjectUUID)

	return multistep.ActionContinue
}

func (s *stepCreateServer) Cleanup(state multistep.StateBag) {
	client := state.Get("client").(*gsclient.Client)
	ui := state.Get("ui").(packer.Ui)

	if s.serverId != "" {
		ui.Say("Shutdown Server...")
		client.ShutdownServer(context.Background(), s.serverId)
	}

	if s.serverId != "" {
		ui.Say("Destroying server...")
		err := client.DeleteServer(context.TODO(), s.serverId)
		if err != nil {
			ui.Error(fmt.Sprintf(
				"Error destroying server. Please destroy it manually: %s", err))
		}
	}

	if s.storageId != "" {
		ui.Say("Destroying storage...")
		err := client.DeleteStorage(context.TODO(), s.storageId)
		if err != nil {
			ui.Error(fmt.Sprintf(
				"Error destroying storage. Please destroy it manually: %s", err))
		}
	}

	if s.ipID != "" {
		ui.Say("Destroying IP...")
		err := client.DeleteIP(context.TODO(), s.ipID)
		if err != nil {
			ui.Error(fmt.Sprintf(
				"Error destroying ip. Please destroy it manually: %s", err))
		}
	}

	return

}
