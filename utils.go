package main

func convertMcpServersToYaml(mcpServers map[string]interface{}) []interface{} {
	var servers []interface{}

	for name, serverData := range mcpServers {
		if server, ok := serverData.(map[string]interface{}); ok {
			orderedServer := OrderedServer{
				Name:  name,
				Extra: make(map[string]interface{}),
			}
			if cmd, ok := server["command"].(string); ok {
				orderedServer.Command = cmd
			}
			if args, ok := server["args"].([]interface{}); ok {
				stringArgs := make([]string, len(args))
				for i, arg := range args {
					if s, ok := arg.(string); ok {
						stringArgs[i] = s
					}
				}
				orderedServer.Args = stringArgs
			}
			if env, ok := server["env"].(map[string]interface{}); ok {
				envMap := make(map[string]string)
				for k, v := range env {
					if s, ok := v.(string); ok {
						envMap[k] = s
					}
				}
				orderedServer.Env = envMap
			}
			for k, v := range server {
				if k != "command" && k != "args" && k != "env" {
					orderedServer.Extra[k] = v
				}
			}
			servers = append(servers, orderedServer)
		}
	}
	return servers
}

func extractClientServers(config map[string]interface{}) map[string]interface{} {
	servers := make(map[string]interface{})
	clientServers, ok := config["servers"].([]interface{})
	if !ok {
		return servers
	}
	for _, serverData := range clientServers {
		if name, serverConfig, isValid := processSingleServerConfig(serverData); isValid {
			servers[name] = serverConfig
		}
	}
	return servers
}

func processSingleServerConfig(serverData interface{}) (serverName string, serverConfig map[string]interface{}, ok bool) {
	server, isValidMap := serverData.(map[string]interface{})
	if !isValidMap {
		return "", nil, false
	}
	name, hasName := server["name"].(string)
	if !hasName || name == "" {
		return "", nil, false
	}
	serverCopy := make(map[string]interface{})
	for k, v := range server {
		if k != "name" {
			serverCopy[k] = v
		}
	}

	return name, serverCopy, true
}
