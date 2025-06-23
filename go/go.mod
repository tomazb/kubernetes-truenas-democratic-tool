module github.com/yourusername/kubernetes-truenas-democratic-tool

go 1.21

require (
	github.com/gin-gonic/gin v1.9.1
	github.com/prometheus/client_golang v1.17.0
	github.com/spf13/cobra v1.8.0
	github.com/spf13/viper v1.17.0
	k8s.io/api v0.28.4
	k8s.io/apimachinery v0.28.4
	k8s.io/client-go v0.28.4
	sigs.k8s.io/controller-runtime v0.16.3
)

require (
	github.com/go-logr/logr v1.3.0
	github.com/go-resty/resty/v2 v2.11.0
	github.com/gorilla/websocket v1.5.1
	github.com/stretchr/testify v1.8.4
	go.uber.org/zap v1.26.0
	golang.org/x/time v0.5.0
	gopkg.in/yaml.v3 v3.0.1
)