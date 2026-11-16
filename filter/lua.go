package filter

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Hsn723/certspotter-client/api"
	lua "github.com/yuin/gopher-lua"
)

const (
	luaExt        = ".lua"
	luaFilterFunc = "filter"
	luaMaxDepth   = 8
)

func isLuaFilter(path string) bool {
	return strings.EqualFold(filepath.Ext(path), luaExt)
}

func applyLuaFilter(filter string, issuances []api.Issuance) ([]api.Issuance, error) {
	b, err := json.Marshal(issuances)
	if err != nil {
		return issuances, err
	}
	var generic interface{}
	if err := json.Unmarshal(b, &generic); err != nil {
		return issuances, err
	}

	L := lua.NewState()
	defer L.Close()
	ctx, cancel := context.WithTimeout(context.Background(), pluginTimeout)
	defer cancel()
	L.SetContext(ctx)

	if err := L.DoFile(filter); err != nil {
		return issuances, err
	}
	filterFunc := L.GetGlobal(luaFilterFunc)
	if filterFunc.Type() != lua.LTFunction {
		return issuances, fmt.Errorf("filter function %q not found in %s", luaFilterFunc, filter)
	}

	arg := toLua(L, generic)
	if arg == lua.LNil {
		return issuances, errors.New("failed to convert issuances to Lua table")
	}
	p := lua.P{
		Fn:      filterFunc,
		NRet:    1,
		Protect: true,
	}
	if err := L.CallByParam(p, arg); err != nil {
		return issuances, err
	}
	ret := L.Get(-1)
	L.Pop(1)
	if _, ok := ret.(*lua.LTable); !ok {
		return issuances, fmt.Errorf("filter function %q in %s did not return a table, got %s", luaFilterFunc, filter, ret.Type())
	}
	out, err := fromLua(ret, 0)
	if err != nil {
		return issuances, err
	}
	if out == nil {
		return []api.Issuance{}, nil
	}
	b, err = json.Marshal(out)
	if err != nil {
		return issuances, err
	}
	var filtered []api.Issuance
	if err := json.Unmarshal(b, &filtered); err != nil {
		return issuances, err
	}
	return filtered, nil
}

func toLua(L *lua.LState, v interface{}) lua.LValue {
	switch t := v.(type) {
	case bool:
		return lua.LBool(t)
	case float64:
		return lua.LNumber(t)
	case string:
		return lua.LString(t)
	case []interface{}:
		tbl := L.CreateTable(len(t), 0)
		for _, e := range t {
			tbl.Append(toLua(L, e))
		}
		return tbl
	case map[string]interface{}:
		tbl := L.CreateTable(0, len(t))
		for k, e := range t {
			tbl.RawSetString(k, toLua(L, e))
		}
		return tbl
	default:
		return lua.LNil
	}
}

func fromLua(v lua.LValue, depth int) (interface{}, error) {
	if depth > luaMaxDepth {
		return nil, fmt.Errorf("exceeded maximum depth of %d", luaMaxDepth)
	}
	switch t := v.(type) {
	case *lua.LNilType:
		return nil, nil
	case lua.LBool:
		return bool(t), nil
	case lua.LNumber:
		return float64(t), nil
	case lua.LString:
		return string(t), nil
	case *lua.LTable:
		return tableFromLua(t, depth)
	default:
		return nil, fmt.Errorf("unsupported lua type %s", v.Type())
	}
}

func tableFromLua(tbl *lua.LTable, depth int) (interface{}, error) {
	if n := tbl.MaxN(); n > 0 {
		arr := make([]interface{}, 0, n)
		for i := 1; i <= n; i++ {
			e := tbl.RawGetInt(i)
			if e == lua.LNil {
				return nil, fmt.Errorf("table has nil value at index %d", i)
			}
			f, err := fromLua(e, depth+1)
			if err != nil {
				return nil, err
			}
			arr = append(arr, f)
		}
		return arr, nil
	}
	var err error
	obj := make(map[string]interface{})
	tbl.ForEach(func(k, v lua.LValue) {
		if err != nil {
			return
		}
		s, ok := k.(lua.LString)
		if !ok {
			err = fmt.Errorf("table has non-string key, type: %s", k.Type())
			return
		}
		var c interface{}
		c, err = fromLua(v, depth+1)
		if err != nil {
			return
		}
		obj[string(s)] = c
	})
	if err != nil {
		return nil, err
	}
	if len(obj) == 0 {
		return nil, nil
	}
	return obj, nil
}
