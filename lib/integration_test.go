package lib

import (
	"context"
	"fmt"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/dsref"
	dsrefspec "github.com/qri-io/qri/dsref/spec"
	"github.com/qri-io/qri/registry"
	"github.com/qri-io/qri/registry/regserver"
	"github.com/qri-io/qri/remote"
	"github.com/qri-io/qri/repo/gen"
	reporef "github.com/qri-io/qri/repo/ref"
	repotest "github.com/qri-io/qri/repo/test"
)

func TestTwoActorRegistryIntegration(t *testing.T) {
	tr := NewNetworkIntegrationTestRunner(t, "integration_two_actor_registry")
	defer tr.Cleanup()

	nasim := tr.InitNasim(t)

	// - nasim creates a dataset
	ref := InitWorldBankDataset(t, nasim)

	// - nasim publishes to the registry
	PublishToRegistry(t, nasim, ref.AliasString())

	if err := AssertLogsEqual(nasim, tr.RegistryInst, ref); err != nil {
		t.Error(err)
	}

	p := &ListParams{}
	refs := ""
	if err := NewDatasetMethods(tr.RegistryInst).ListRawRefs(p, &refs); err != nil {
		t.Fatal(err)
	}
	t.Log(refs)

	hinshun := tr.InitHinshun(t)

	// - hinshun searches the registry for nasim's dataset name, gets a result
	if results := SearchFor(t, hinshun, "bank"); len(results) < 1 {
		t.Logf("expected at least one result in registry search")
		// t.Errorf("expected at least one result in registry search")
	}

	// - hunshun fetches a preview of nasim's dataset
	// TODO (b5) - need to use the ref returned from search results
	t.Log(ref.String())
	Preview(t, hinshun, ref.String())

	// - hinshun clones nasim's dataset
	Clone(t, hinshun, ref.AliasString())

	if err := AssertLogsEqual(nasim, hinshun, ref); err != nil {
		t.Error(err)
	}

	// 5. nasim commits a new version
	ref = Commit2WorldBank(t, nasim)

	// 6. nasim re-publishes to the registry
	PublishToRegistry(t, nasim, ref.AliasString())

	// 7. hinshun logsyncs with the registry for world bank dataset, sees multiple versions
	dsm := NewDatasetMethods(hinshun)
	res := &reporef.DatasetRef{}
	if err := dsm.Add(&AddParams{LogsOnly: true, Ref: ref.String()}, res); err != nil {
		t.Errorf("cloning logs: %s", err)
	}

	if err := AssertLogsEqual(nasim, hinshun, ref); err != nil {
		t.Error(err)
	}

	// TODO (b5) - assert hinshun DOES NOT have blocks for the latest commit to world bank dataset

	// 8. hinshun clones latest version
	Clone(t, hinshun, ref.AliasString())

	// TODO (b5) - assert hinshun has world bank dataset blocks

	// all three should now have the same HEAD reference & InitID
	dsrefspec.ConsistentResolvers(t, dsref.Ref{
		Username: ref.Peername,
		Name:     ref.Name,
	},
		nasim.Repo(),
		hinshun.Repo(),
		tr.RegistryInst.Repo(),
	)
}

func TestAddCheckoutIntegration(t *testing.T) {
	tr := NewNetworkIntegrationTestRunner(t, "integration_add_checkout")
	defer tr.Cleanup()

	nasim := tr.InitNasim(t)

	// - nasim creates a dataset, publishes to registry
	ref := InitWorldBankDataset(t, nasim)
	PublishToRegistry(t, nasim, ref.AliasString())

	hinshun := tr.InitHinshun(t)
	dsm := NewDatasetMethods(hinshun)
	res := &reporef.DatasetRef{}

	checkoutPath := filepath.Join(tr.hinshunRepo.RootPath, "wbp")
	err := dsm.Add(&AddParams{
		Ref:     ref.String(),
		LinkDir: checkoutPath,
	}, res)
	if err != nil {
		t.Errorf("adding with linked directory: %s", err)
	}

}

func TestReferencePulling(t *testing.T) {
	tr := NewNetworkIntegrationTestRunner(t, "integration_reference_pulling")
	defer tr.Cleanup()

	nasim := tr.InitNasim(t)

	// - nasim creates a dataset, publishes to registry
	ref := InitWorldBankDataset(t, nasim)
	PublishToRegistry(t, nasim, ref.AliasString())

	hinshun := tr.InitHinshun(t)
	sqlm := NewSQLMethods(hinshun)

	// fetch this from the registry by default
	p := &SQLQueryParams{
		Query:        "SELECT * FROM nasim/world_bank_population a LIMIT 1",
		OutputFormat: "json",
	}
	results := make([]byte, 0)
	if err := sqlm.Exec(p, &results); err != nil {
		t.Fatal(err)
	}

	// re-run. dataset should now be local, and no longer require registry to
	// resolve
	p = &SQLQueryParams{
		Query:        "SELECT * FROM nasim/world_bank_population a LIMIT 1 OFFSET 1",
		OutputFormat: "json",
		ResolverMode: "local",
	}
	results = make([]byte, 0)
	if err := sqlm.Exec(p, &results); err != nil {
		t.Fatal(err)
	}

	// create adnan
	adnan := tr.InitAdnan(t)
	dsm := NewDatasetMethods(adnan)

	// run a transform script that relies on world_bank_population, which adnan's
	// node should automatically pull to execute thisscript
	tfScriptData := `
wbp = load_dataset("nasim/world_bank_population")

def transform(ds, ctx):
	body = wbp.get_body() + [["g","h","i",False,3]]
	ds.set_body(body)
`
	scriptPath, err := tr.adnanRepo.WriteRootFile("transform.star", tfScriptData)
	if err != nil {
		t.Fatal(err)
	}

	saveParams := &SaveParams{
		Ref: "me/wbp_plus_one",
		FilePaths: []string{
			scriptPath,
		},
	}
	res := &reporef.DatasetRef{}
	if err := dsm.Save(saveParams, res); err != nil {
		t.Fatal(err)
	}
}

type NetworkIntegrationTestRunner struct {
	Ctx        context.Context
	prefix     string
	TestCrypto gen.CryptoGenerator

	nasimRepo, hinshunRepo, adnanRepo *repotest.TempRepo
	Nasim, Hinshun, Adnan             *Instance

	registryRepo       *repotest.TempRepo
	Registry           registry.Registry
	RegistryInst       *Instance
	RegistryHTTPServer *httptest.Server
}

func NewNetworkIntegrationTestRunner(t *testing.T, prefix string) *NetworkIntegrationTestRunner {
	tr := &NetworkIntegrationTestRunner{
		Ctx:        context.Background(),
		prefix:     prefix,
		TestCrypto: repotest.NewTestCrypto(),
	}

	tr.InitRegistry(t)

	return tr
}

func (tr *NetworkIntegrationTestRunner) Cleanup() {
	tr.RegistryHTTPServer.Close()
	tr.registryRepo.Delete()
	if tr.nasimRepo != nil {
		tr.nasimRepo.Delete()
	}
	if tr.hinshunRepo != nil {
		tr.hinshunRepo.Delete()
	}
}

func (tr *NetworkIntegrationTestRunner) InitNasim(t *testing.T) *Instance {
	r, err := repotest.NewTempRepo("nasim", fmt.Sprintf("%s_nasim", tr.prefix), tr.TestCrypto)
	if err != nil {
		t.Fatal(err)
	}

	cfg := r.GetConfig()
	cfg.Registry.Location = tr.RegistryHTTPServer.URL
	r.WriteConfigFile()
	tr.nasimRepo = &r

	if tr.Nasim, err = NewInstance(tr.Ctx, r.QriPath); err != nil {
		t.Fatal(err)
	}

	return tr.Nasim
}

func (tr *NetworkIntegrationTestRunner) InitHinshun(t *testing.T) *Instance {
	r, err := repotest.NewTempRepo("hinshun", fmt.Sprintf("%s_hinshun", tr.prefix), tr.TestCrypto)
	if err != nil {
		t.Fatal(err)
	}

	tr.hinshunRepo = &r
	cfg := tr.hinshunRepo.GetConfig()
	cfg.Registry.Location = tr.RegistryHTTPServer.URL
	tr.hinshunRepo.WriteConfigFile()

	if tr.Hinshun, err = NewInstance(tr.Ctx, tr.hinshunRepo.QriPath); err != nil {
		t.Fatal(err)
	}

	return tr.Hinshun
}

func (tr *NetworkIntegrationTestRunner) InitAdnan(t *testing.T) *Instance {
	r, err := repotest.NewTempRepo("adnan", fmt.Sprintf("%s_adnan", tr.prefix), tr.TestCrypto)
	if err != nil {
		t.Fatal(err)
	}

	cfg := r.GetConfig()
	cfg.Registry.Location = tr.RegistryHTTPServer.URL
	r.WriteConfigFile()
	tr.adnanRepo = &r

	if tr.Adnan, err = NewInstance(tr.Ctx, r.QriPath); err != nil {
		t.Fatal(err)
	}

	return tr.Adnan
}

func (tr *NetworkIntegrationTestRunner) InitRegistry(t *testing.T) {
	rr, err := repotest.NewTempRepo("registry", fmt.Sprintf("%s_registry", tr.prefix), tr.TestCrypto)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("registry qri path: %s", rr.QriPath)

	tr.registryRepo = &rr

	cfg := rr.GetConfig()
	cfg.Registry.Location = ""
	cfg.Remote = &config.Remote{
		Enabled:          true,
		AcceptSizeMax:    -1,
		AcceptTimeoutMs:  -1,
		RequireAllBlocks: false,
		AllowRemoves:     true,
	}

	rr.WriteConfigFile()

	opts := []Option{
		OptSetIPFSPath(rr.IPFSPath),
	}

	tr.RegistryInst, err = NewInstance(tr.Ctx, rr.QriPath, opts...)
	if err != nil {
		t.Fatal(err)
	}

	node := tr.RegistryInst.Node()
	if node == nil {
		t.Fatal("creating a Registry for NetworkIntegration test fails if `qri connect` is running")
	}

	rem, err := remote.NewRemote(node, cfg.Remote)
	if err != nil {
		t.Fatal(err)
	}

	tr.Registry = registry.Registry{
		Remote:   rem,
		Profiles: registry.NewMemProfiles(),
		Search:   regserver.MockRepoSearch{Repo: tr.RegistryInst.Repo()},
	}

	_, tr.RegistryHTTPServer = regserver.NewMockServerRegistry(tr.Registry)
}

func AssertLogsEqual(a, b *Instance, ref *reporef.DatasetRef) error {
	r := reporef.ConvertToDsref(*ref)

	aLogs, err := a.logbook.DatasetRef(context.Background(), r)
	if err != nil {
		return fmt.Errorf("fetching logs for a instance: %s", err)
	}

	bLogs, err := b.logbook.DatasetRef(context.Background(), r)
	if err != nil {
		return fmt.Errorf("fetching logs for b instance: %s", err)
	}

	if aLogs.ID() != bLogs.ID() {
		return fmt.Errorf("log ID mismatch. %s != %s", aLogs.ID(), bLogs.ID())
	}

	if len(aLogs.Logs) != len(bLogs.Logs) {
		return fmt.Errorf("oplength mismatch. %d != %d", len(aLogs.Logs), len(bLogs.Logs))
	}

	return nil
}

func InitWorldBankDataset(t *testing.T, inst *Instance) *reporef.DatasetRef {
	res := &reporef.DatasetRef{}
	err := NewDatasetMethods(inst).Save(&SaveParams{
		Publish: true,
		Ref:     "me/world_bank_population",
		Dataset: &dataset.Dataset{
			Meta: &dataset.Meta{
				Title: "World Bank Population",
			},
			BodyPath: "body.csv",
			BodyBytes: []byte(`a,b,c,true,2
d,e,f,false,3`),
			Readme: &dataset.Readme{
				ScriptPath:  "readme.md",
				ScriptBytes: []byte("#World Bank Population\nhow many people live on this planet?"),
			},
		},
	}, res)

	if err != nil {
		log.Fatalf("saving dataset version: %s", err)
	}

	return res
}

func Commit2WorldBank(t *testing.T, inst *Instance) *reporef.DatasetRef {
	res := &reporef.DatasetRef{}
	err := NewDatasetMethods(inst).Save(&SaveParams{
		Publish: true,
		Ref:     "me/world_bank_population",
		Dataset: &dataset.Dataset{
			Meta: &dataset.Meta{
				Title: "World Bank Population",
			},
			BodyPath: "body.csv",
			BodyBytes: []byte(`a,b,c,true,2
d,e,f,false,3
g,g,i,true,4`),
		},
	}, res)

	if err != nil {
		log.Fatalf("saving dataset version: %s", err)
	}

	return res
}

func PublishToRegistry(t *testing.T, inst *Instance, refstr string) *dsref.Ref {
	res := dsref.Ref{}
	err := NewRemoteMethods(inst).Publish(&PublicationParams{
		Ref: refstr,
	}, &res)

	if err != nil {
		log.Fatalf("publishing dataset: %s", err)
	}

	return &res
}

func SearchFor(t *testing.T, inst *Instance, term string) []SearchResult {
	results := []SearchResult{}
	if err := NewSearchMethods(inst).Search(&SearchParams{QueryString: term}, &results); err != nil {
		t.Fatal(err)
	}

	return results
}

func Clone(t *testing.T, inst *Instance, refstr string) *reporef.DatasetRef {
	res := &reporef.DatasetRef{}
	if err := NewDatasetMethods(inst).Add(&AddParams{Ref: refstr}, res); err != nil {
		t.Fatalf("cloning dataset %s: %s", refstr, err)
	}
	return res
}

func Preview(t *testing.T, inst *Instance, refstr string) *dataset.Dataset {
	p := &PreviewParams{
		Ref: refstr,
	}
	res := &dataset.Dataset{}
	if err := NewRemoteMethods(inst).Preview(p, res); err != nil {
		t.Fatal(err)
	}
	return res
}
