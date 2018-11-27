package main

import (
	"flag"
	"github.com/mitchellh/go-homedir"
	"net/http"
	"os"
	"strings"
	"time"

	core_v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/log"
)

const rootDoc = `<html>
<head><title>Service Target Pods Num Exporter</title></head>
<body>
<h1>Service Target Pods Num Exporter</h1>
<p><a href="/metrics">Metrics</a></p>
</body>
</html>
`

const (
	defaultInterval = 30
	defaultAddr     = ":9299"
)

var interval = flag.Int("scrapeInterval", defaultInterval, "Interval to scrape status.")
var addr = flag.String("listenAddress", defaultAddr, "The address to listen on for HTTP requests.")

var labels = []string{
	"service",
	"namespace",
}

var (
	serviceTargetPodsNum = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "kube_service_target_pods_num",
			Help: "Target pods number of service.",
		},
		labels,
	)
)

var kubeClient = func() kubernetes.Interface {
	var ret kubernetes.Interface
	config, err := rest.InClusterConfig()
	if err != nil {
		var kubeconfigPath string
		if os.Getenv("KUBECONFIG") == "" {
			home, err := homedir.Dir()
			if err != nil {
				panic(err)
			}
			kubeconfigPath = home + "/.kube/config"
		} else {
			kubeconfigPath = os.Getenv("KUBECONFIG")
		}
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
		if err != nil {
			panic(err)
		}
	}
	ret, err = kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}
	return ret
}()

func init() {
	prometheus.MustRegister(serviceTargetPodsNum)
}

func getServiceList() ([]core_v1.Service, error) {
	out, err := kubeClient.CoreV1().Services("").List(meta_v1.ListOptions{})
	return out.Items, err
}

func targetPodsNum(namespace string, selector map[string]string) int {
	ls := ""
	for k, v := range selector {
		ls = ls + k + "=" + v + ","
	}
	ls = strings.TrimRight(ls, ",")
	o := meta_v1.ListOptions{
		LabelSelector: ls,
	}
	if out, err := kubeClient.CoreV1().Pods(namespace).List(o); err != nil {
		log.Errorln(err)
		return 0
	} else {
		ret := 0
		for _, p := range out.Items {
			if p.Status.Phase == core_v1.PodRunning {
				ret++
			}
		}
		return ret
	}
}

func main() {
	flag.Parse()
	go func() {
		for {
			o, e := getServiceList()
			if e != nil {
				log.Errorln(e)
			}
			for _, s := range o {
				if len(s.Spec.Selector) != 0 {
					p := targetPodsNum(s.ObjectMeta.Namespace, s.Spec.Selector)
					label := prometheus.Labels{
						"service":   s.ObjectMeta.Name,
						"namespace": s.ObjectMeta.Namespace,
					}
					serviceTargetPodsNum.With(label).Set(float64(p))
				}
			}
			time.Sleep(time.Duration(*interval) * time.Second)
		}
	}()
	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(rootDoc))
	})

	log.Fatal(http.ListenAndServe(*addr, nil))
}
