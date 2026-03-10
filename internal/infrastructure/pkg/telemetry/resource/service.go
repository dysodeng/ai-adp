package resource

import "github.com/dysodeng/ai-adp/internal/infrastructure/config"

func ServiceName() string {
	name := config.GlobalConfig.App.Name
	if config.GlobalConfig.Monitor.ServiceName != "" {
		name = config.GlobalConfig.Monitor.ServiceName
	}
	return name
}
