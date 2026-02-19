package etcd

import (
	"context"

	clientv3 "go.etcd.io/etcd/client/v3"
)

type CasResult struct {
	Applied bool
}

func compareAndSwap(
	ctx context.Context,
	client *clientv3.Client,
	key string,
	expectedModRevision int64,
	expectedValue string,
	newValue string,
	thenOps ...clientv3.Op,
) (CasResult, error) {
	ops := make([]clientv3.Op, 0, len(thenOps)+1)
	ops = append(ops, clientv3.OpPut(key, newValue))
	ops = append(ops, thenOps...)

	resp, err := client.Txn(ctx).
		If(
			clientv3.Compare(clientv3.ModRevision(key), "=", expectedModRevision),
			clientv3.Compare(clientv3.Value(key), "=", expectedValue),
		).
		Then(ops...).
		Commit()
	if err != nil {
		return CasResult{}, err
	}
	return CasResult{Applied: resp.Succeeded}, nil
}
