package api

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/base/dsfs"
	"github.com/qri-io/qri/lib"
	"github.com/qri-io/qri/p2p"
	reporef "github.com/qri-io/qri/repo/ref"
)

type APITestRunner struct {
	Node         *p2p.QriNode
	NodeTeardown func()
	Inst         *lib.Instance
	DsfsTsFunc   func() time.Time
	TmpDir       string
	WorkDir      string
	PrevXformVer string
}

func NewAPITestRunner(t *testing.T) *APITestRunner {
	run := APITestRunner{}
	run.Node, run.NodeTeardown = newTestNode(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	run.Inst = newTestInstanceWithProfileFromNode(ctx, run.Node)

	tmpDir, err := ioutil.TempDir("", "api_test")
	if err != nil {
		t.Fatal(err)
	}
	run.TmpDir = tmpDir

	counter := 0
	run.DsfsTsFunc = dsfs.Timestamp
	dsfs.Timestamp = func() time.Time {
		counter++
		return time.Date(2001, 01, 01, 01, counter, 01, 01, time.UTC)
	}

	run.PrevXformVer = APIVersion
	APIVersion = "test_version"

	return &run
}

func (r *APITestRunner) Delete() {
	os.RemoveAll(r.TmpDir)
	APIVersion = r.PrevXformVer
	r.NodeTeardown()
}

func (r *APITestRunner) MustMakeWorkDir(t *testing.T, name string) string {
	r.WorkDir = filepath.Join(r.TmpDir, name)
	if err := os.MkdirAll(r.WorkDir, os.ModePerm); err != nil {
		t.Fatal(err)
	}
	return r.WorkDir
}

func (r *APITestRunner) BuildDataset(dsName string) *dataset.Dataset {
	ds := dataset.Dataset{
		Peername: "peer",
		Name:     dsName,
	}
	return &ds
}

func (r *APITestRunner) SaveDataset(ds *dataset.Dataset, bodyFilename string) {
	dsm := lib.NewDatasetMethods(r.Inst)
	saveParams := lib.SaveParams{
		Ref:      fmt.Sprintf("peer/%s", ds.Name),
		Dataset:  ds,
		BodyPath: bodyFilename,
	}
	res := reporef.DatasetRef{}
	if err := dsm.Save(&saveParams, &res); err != nil {
		panic(err)
	}
}

func (r *APITestRunner) NewRenderHandlers() *RenderHandlers {
	return NewRenderHandlers(r.Inst)
}
