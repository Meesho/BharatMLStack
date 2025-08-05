package network

import "hash/fnv"

type RequestRouter struct {
	clients []*ClientManager
}

func NewRequestRouter(numClients int, serverPort int) *RequestRouter {
	clients := make([]*ClientManager, numClients)
	for i := 0; i < numClients; i++ {
		clients[i] = NewClientManager(serverPort)
	}
	return &RequestRouter{
		clients: clients,
	}
}

func (r *RequestRouter) GetData(key string, ip string) chan ResponseMessage {
	hash := fnv.New32()
	hash.Write([]byte(key))
	hashValue := hash.Sum32()
	return r.clients[hashValue%uint32(len(r.clients))].GetData(key, ip)
}
