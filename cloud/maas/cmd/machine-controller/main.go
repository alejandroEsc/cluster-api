package machine_controller

import (
	"encoding/json"

	"github.com/spf13/viper"
	"github.com/lxc/lxd/shared/logger"
	"github.com/juju/gomaasapi"
	"github.com/golang/glog"
	m "github.com/alejandroEsc/maas-cli/pkg/maas"
	"github.com/kubernetes-incubator/apiserver-builder/pkg/controller"
	"os"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/client-go/tools/record"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/uuid"
	"sigs.k8s.io/cluster-api/pkg/controller/config"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/leaderelection"
	"sigs.k8s.io/cluster-api/pkg/client/clientset_generated/clientset"

	"sigs.k8s.io/cluster-api/pkg/controller/machine"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/cluster-api/pkg/controller/sharedinformers"
	v1core "k8s.io/client-go/kubernetes/typed/core/v1"
	clusterapiclientsetscheme "sigs.k8s.io/cluster-api/pkg/client/clientset_generated/clientset/scheme"

	"sigs.k8s.io/cluster-api/cloud/google/machinesetup"
	"sigs.k8s.io/cluster-api/cloud/maas"

)

const (
	machineControllerName = "Maas-Provider-Machine-Controller"
	apiKeyKey         = "api_key"
	maasURLKey        = "url"
	maasAPIVersionKey = "api_version"
	machineSetupConfigsPath = "machine_setup_configs_path"
)

// Init initializes the environment variables to be used by the app
func Init() {
	viper.AutomaticEnv()
	viper.SetEnvPrefix("maas_client")
	viper.BindEnv(maasURLKey)
	viper.BindEnv(apiKeyKey)
	viper.BindEnv(maasAPIVersionKey)
	viper.BindEnv()
}

func main() {
	Init()
	apiKey := viper.GetString(apiKeyKey)
	maasURL := viper.GetString(maasURLKey)
	apiVersion := viper.GetString(maasAPIVersionKey)
	machineSetupConfigsPath := viper.GetString(machineSetupConfigsPath)

	glog.Infof("%s %s %s", apiKey, maasURL, apiVersion)

	logger.Infof("Starting Sample-MAAS Client...")

	// Create API server endpoint.
	authClient, err := gomaasapi.NewAuthenticatedClient(gomaasapi.AddAPIVersionToURL(maasURL, apiVersion), apiKey)
	checkError(err)
	maas := gomaasapi.NewMAAS(*authClient)

	maasCLI := m.NewMaas(maas)

	getMAASVersion(maasCLI)

	server := NewMachineControllerServer(machineSetupConfigsPath, maasCLI)

	err = runMachineController(server)
	if err != nil{
		glog.Errorf("error running machine controller: %s", err)
	}


}

func runMachineController(server *MachineControllerServer) error {
	kubeConfig, err := controller.GetConfig(server.CommonConfig.Kubeconfig)
	if err != nil {
		glog.Errorf("Could not create Config for talking to the apiserver: %v", err)
		return err
	}

	kubeClientControl, err := kubernetes.NewForConfig(
		rest.AddUserAgent(kubeConfig, "machine-controller-manager"),
	)
	if err != nil {
		glog.Errorf("Invalid API configuration for kubeconfig-control: %v", err)
		return err
	}

	recorder, err := createRecorder(kubeClientControl)
	if err != nil {
		glog.Errorf("Could not create event recorder : %v", err)
		return err
	}

	// run function will block and never return.
	run := func(stop <-chan struct{}) {
		startMachineController(server, stop)
	}

	leaderElectConfig := config.GetLeaderElectionConfig()
	if !leaderElectConfig.LeaderElect {
		run(make(<-chan (struct{})))
	}

	// Identity used to distinguish between multiple machine controller instances.
	id, err := os.Hostname()
	if err != nil {
		return err
	}

	leaderElectionClient := kubernetes.NewForConfigOrDie(rest.AddUserAgent(kubeConfig, "machine-leader-election"))

	id = id + "-" + string(uuid.NewUUID())

	// Lock required for leader election
	rl, err := resourcelock.New(
		leaderElectConfig.ResourceLock,
		metav1.NamespaceSystem,
		machineControllerName,
		leaderElectionClient.CoreV1(),
		resourcelock.ResourceLockConfig{
			Identity:      id + "-" + machineControllerName,
			EventRecorder: recorder,
		})
	if err != nil {
		return err
	}

	// Try and become the leader and start machine controller loops
	leaderelection.RunOrDie(leaderelection.LeaderElectionConfig{
		Lock:          rl,
		LeaseDuration: leaderElectConfig.LeaseDuration.Duration,
		RenewDeadline: leaderElectConfig.RenewDeadline.Duration,
		RetryPeriod:   leaderElectConfig.RetryPeriod.Duration,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: run,
			OnStoppedLeading: func() {
				glog.Fatalf("leaderelection lost")
			},
		},
	})
	panic("unreachable")


}

func startMachineController(server *MachineControllerServer, shutdown <-chan struct{}) {
	config, err := controller.GetConfig(server.CommonConfig.Kubeconfig)
	if err != nil {
		glog.Fatalf("Could not create Config for talking to the apiserver: %v", err)
	}

	client, err := clientset.NewForConfig(config)
	if err != nil {
		glog.Fatalf("Could not create client for talking to the apiserver: %v", err)
	}

	configWatch, err := machinesetup.NewConfigWatch(server.MachineSetupConfigsPath)
	if err != nil {
		glog.Fatalf("Could not create config watch: %v", err)
	}
	params := maas.MachineActuatorParams{
		V1Alpha1Client:           client.ClusterV1alpha1(),
		MachineSetupConfigGetter: configWatch,
	}
	actuator, err := maas.NewMachineActuator(params)

	if err != nil {
		glog.Fatalf("Could not create Google machine actuator: %v", err)
	}

	si := sharedinformers.NewSharedInformers(config, shutdown)
	// If this doesn't compile, the code generator probably
	// overwrote the customized NewMachineController function.
	c := machine.NewMachineController(config, si, actuator)
	c.RunAsync(shutdown)
}



func getMAASVersion(maasCLI *m.Maas) {
	version, err := maasCLI.GetMAASVersion()
	checkError(err)
	jp, err := json.MarshalIndent(version, "", "\t")
	checkError(err)
	logger.Infof("\n%s", jp)
}

func checkError(err error) {
	if err != nil {
		glog.Errorf(err.Error())
	}
}

func checkErrorMsg(err error, msg string) {
	if err != nil {
		glog.Errorf("%s, %s", msg, err.Error())
	}
}

func createRecorder(kubeClient *kubernetes.Clientset) (record.EventRecorder, error) {

	eventsScheme := runtime.NewScheme()
	if err := corev1.AddToScheme(eventsScheme); err != nil {
		return nil, err
	}
	// We also emit events for our own types
	clusterapiclientsetscheme.AddToScheme(eventsScheme)

	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(glog.Infof)
	eventBroadcaster.StartRecordingToSink(&v1core.EventSinkImpl{Interface: v1core.New(kubeClient.CoreV1().RESTClient()).Events("")})
	return eventBroadcaster.NewRecorder(eventsScheme, corev1.EventSource{Component: machineControllerName}), nil
}