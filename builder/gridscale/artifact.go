package gridscale

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/gridscale/gsclient-go"
)

type Artifact struct {
	// The name of the snapshot
	SnapshotName string

	// The ID of the image
	SnapshotId string

	// The name of the region
	RegionNames []string

	// The client for making API calls
	Client *gsclient.Client
}

func (*Artifact) BuilderId() string {
	return BuilderId
}

func (*Artifact) Files() []string {
	// No files with DigitalOcean
	return nil
}

func (a *Artifact) Id() string {
	return fmt.Sprintf("%s:%s", strings.Join(a.RegionNames[:], ","), a.SnapshotId)
}

func (a *Artifact) String() string {
	return fmt.Sprintf("A snapshot was created: '%v' (ID: %v) in regions '%v'", a.SnapshotName, a.SnapshotId, strings.Join(a.RegionNames[:], ","))
}

func (a *Artifact) State(name string) interface{} {
	return nil
}

func (a *Artifact) Destroy() error {
	log.Printf("Destroying image: %d (%s)", a.SnapshotId, a.SnapshotName)
	err := a.Client.DeleteTemplate(context.Background(), a.SnapshotId)
	return err
}
