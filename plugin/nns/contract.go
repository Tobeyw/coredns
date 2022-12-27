package nns

import (
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/nspcc-dev/neo-go/pkg/rpcclient/nns"
	"math/big"

	"github.com/nspcc-dev/neo-go/pkg/neorpc/result"
	"github.com/nspcc-dev/neo-go/pkg/rpcclient"
	"github.com/nspcc-dev/neo-go/pkg/smartcontract"
	"github.com/nspcc-dev/neo-go/pkg/util"
	"github.com/nspcc-dev/neo-go/pkg/vm/stackitem"
)

type nnsRecord struct {
	Name string
	Type nns.RecordType
	Data string
}

func resolve(rpc *rpcclient.Client, hash util.Uint160, name string, nnsType nns.RecordType) (string, error) {
	res, err := rpc.InvokeFunction(hash, "resolve", []smartcontract.Parameter{
		{
			Type:  smartcontract.StringType,
			Value: name,
		},
		{
			Type:  smartcontract.IntegerType,
			Value: big.NewInt(int64(nnsType)),
		},
	}, nil)
	version, _ := rpc.GetVersion()

	chash := hash.String()
	fmt.Println(rpc, chash, version)
	if err != nil {
		return "", err
	}
	log.Info(hash, res, int64(nnsType))
	if err = getInvocationError(res); err != nil {
		return "", err
	}

	return getString(res.Stack)
}

func getAllRecords(rpc *rpcclient.Client, hash util.Uint160, name string) ([]nnsRecord, error) {
	res, err := rpc.InvokeFunction(hash, "getAllRecords", []smartcontract.Parameter{
		{
			Type:  smartcontract.StringType,
			Value: name,
		},
	}, nil)
	if err != nil {
		return nil, err
	}

	if err = getInvocationError(res); err != nil {
		return nil, err
	}

	return getRecordsIterator(rpc, res.Session, res.Stack)
}

func getRecords(rpc *rpcclient.Client, hash util.Uint160, name string, nnsType nns.RecordType) ([]string, error) {
	res, err := rpc.InvokeFunction(hash, "getRecords", []smartcontract.Parameter{
		{
			Type:  smartcontract.StringType,
			Value: name,
		},
		{
			Type:  smartcontract.IntegerType,
			Value: big.NewInt(int64(nnsType)),
		},
	}, nil)
	if err != nil {
		return nil, err
	}

	if err = getInvocationError(res); err != nil {
		return nil, err
	}

	return getArrString(res.Stack)
}

func getRecord(rpc *rpcclient.Client, hash util.Uint160, name string, nnsType nns.RecordType) ([]string, error) {
	res, err := rpc.InvokeFunction(hash, "getRecord", []smartcontract.Parameter{
		{
			Type:  smartcontract.StringType,
			Value: name,
		},
		{
			Type:  smartcontract.IntegerType,
			Value: big.NewInt(int64(nnsType)),
		},
	}, nil)
	if err != nil {
		return nil, err
	}

	if err = getInvocationError(res); err != nil {
		return nil, err
	}

	return getArrString(res.Stack)
}

func getInvocationError(result *result.Invoke) error {
	if result.State != "HALT" {
		return fmt.Errorf("invocation failed: %s", result.FaultException)
	}
	if len(result.Stack) == 0 {
		return errors.New("result stack is empty")
	}
	return nil
}

func getString(st []stackitem.Item) (string, error) {
	index := len(st) - 1 // top stack element is last in the array
	arr, err := st[index].Convert(stackitem.ByteArrayT)
	if err != nil {
		return "", err
	}
	if _, ok := arr.(stackitem.Null); ok {
		return "", nil
	}
	dd := arr.Value().([]uint8)
	fmt.Println(dd)

	res := B2S(dd)
	return res, nil
}
func B2S(bs []uint8) string {
	ba := []byte{}
	for _, b := range bs {
		ba = append(ba, byte(b))
	}
	return string(ba)
}
func getArrString(st []stackitem.Item) ([]string, error) {
	index := len(st) - 1 // top stack element is last in the array
	arr, err := st[index].Convert(stackitem.ArrayT)
	if err != nil {
		return nil, err
	}
	if _, ok := arr.(stackitem.Null); ok {
		return nil, nil
	}

	iterator, ok := arr.Value().([]stackitem.Item)
	if !ok {
		return nil, errors.New("bad conversion")
	}

	res := make([]string, len(iterator))
	for i, item := range iterator {
		bs, err := item.TryBytes()
		if err != nil {
			return nil, err
		}
		res[i] = string(bs)
	}

	return res, nil
}

func getRecordsIterator(rpc *rpcclient.Client, sessionId uuid.UUID, st []stackitem.Item) ([]nnsRecord, error) {

	index := len(st) - 1 // top stack element is last in the array
	tmp, err := st[index].Convert(stackitem.InteropT)

	if err != nil {
		return nil, err
	}
	iterator, _ := tmp.Value().(result.Iterator)

	iteratorId := *iterator.ID
	res, err := rpc.TraverseIterator(sessionId, iteratorId, 10)

	result := make([]nnsRecord, len(res))
	for i, item := range res {
		structArr, ok := item.Value().([]stackitem.Item)
		if !ok {
			return nil, errors.New("bad conversion")
		}
		if len(structArr) != 3 {
			return nil, errors.New("invalid response struct")
		}

		nameBytes, err := structArr[0].TryBytes()
		if err != nil {
			return nil, err
		}
		integer, err := structArr[1].TryInteger()
		if err != nil {
			return nil, err
		}
		typeBytes := integer.Bytes()
		if len(typeBytes) != 1 {
			return nil, errors.New("invalid nns type")
		}

		dataBytes, err := structArr[2].TryBytes()
		if err != nil {
			return nil, err
		}

		result[i] = nnsRecord{
			Name: string(nameBytes),
			Type: nns.RecordType(typeBytes[0]),
			Data: string(dataBytes),
		}
	}

	return result, nil
}
