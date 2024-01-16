package discovery_test

import (
	"context"
	"cto-github.cisco.com/NFV-BU/go-lanai/pkg/bootstrap"
	"cto-github.cisco.com/NFV-BU/go-lanai/pkg/consul"
	"cto-github.cisco.com/NFV-BU/go-lanai/pkg/discovery"
	discoveryinit "cto-github.cisco.com/NFV-BU/go-lanai/pkg/discovery/init"
	"cto-github.cisco.com/NFV-BU/go-lanai/pkg/discovery/init/testdata"
	"cto-github.cisco.com/NFV-BU/go-lanai/test"
	"cto-github.cisco.com/NFV-BU/go-lanai/test/apptest"
	"cto-github.cisco.com/NFV-BU/go-lanai/test/consultest"
	"cto-github.cisco.com/NFV-BU/go-lanai/test/ittest"
	"github.com/hashicorp/consul/api"
	"github.com/onsi/gomega"
	. "github.com/onsi/gomega"
	"go.uber.org/fx"
	"testing"
)

const TestRegisterFuzzyJsonPathTags = `$.Tags`
const TestServiceID = `testservice-8080-d8755f792d`

/*************************
	Test Setup
 *************************/

func OverrideTestServiceID(customizers *discovery.Customizers) {
	customizers.Add(discovery.CustomizerFunc(func(ctx context.Context, reg *api.AgentServiceRegistration) {
		reg.ID = TestServiceID
		reg.Tags = append(reg.Tags, TestServiceID)
	}))
}

/*************************
	Tests
 *************************/

type TestModuleDI struct {
	fx.In
	Consul              *consul.Connection
	AppContext          *bootstrap.ApplicationContext
	DiscoveryProperties discovery.DiscoveryProperties
	DiscoveryClient     discovery.Client
	Registration        *api.AgentServiceRegistration
	Customizers         *discovery.Customizers
}

func TestModuleInit(t *testing.T) {
	di := TestModuleDI{}
	test.RunTest(context.Background(), t,
		apptest.Bootstrap(),
		consultest.WithHttpPlayback(t,
			//consultest.HttpRecordingMode(),
			// Note: tags may contains build time, should be ignored
			consultest.MoreHTTPVCROptions(ittest.HttpRecordMatching(ittest.FuzzyJsonPaths(
				TestRegisterFuzzyJsonPathTags,
			))),
		),
		apptest.WithBootstrapConfigFS(testdata.TestBootstrapFS),
		apptest.WithConfigFS(testdata.TestApplicationFS),
		apptest.WithModules(discoveryinit.Module),
		apptest.WithFxOptions(
			fx.Invoke(OverrideTestServiceID),
		),
		apptest.WithDI(&di),
		test.GomegaSubTest(SubTestVerifyRegistration(&di), "TestVerifyRegistration"),
	)
}

/*************************
	Sub-Test Cases
 *************************/

func SubTestVerifyRegistration(di *TestModuleDI) test.GomegaSubTestFunc {
	return func(ctx context.Context, t *testing.T, g *gomega.WithT) {
		client := di.Consul.Client()
		catalogs, _, e := client.Catalog().Service("testservice", TestServiceID, (&api.QueryOptions{}).WithContext(ctx))
		g.Expect(e).To(Succeed(), "getting service catalog should not fail")
		g.Expect(catalogs).ToNot(BeEmpty(), "service catalog should not be empty")
		var svc *api.CatalogService
		for i := range catalogs {
			if TestServiceID == catalogs[i].ServiceID {
				svc = catalogs[i]
				break
			}
		}
		g.Expect(svc).ToNot(BeNil(), "service catalog should contain expected instance")
	}
}