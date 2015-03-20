// Copyright © 2013-14 Steve Francia <spf@spf13.com>.
//
// Licensed under the Simple Public License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://opensource.org/licenses/Simple-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tpl

import (
	"bytes"
	"errors"
	"fmt"
	"html"
	"html/template"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/eknkc/amber"
	"github.com/spf13/cast"
	bp "github.com/spf13/hugo/bufferpool"
	"github.com/spf13/hugo/helpers"
	jww "github.com/spf13/jwalterweatherman"
	"github.com/yosssi/ace"
)

var localTemplates *template.Template
var tmpl Template
var funcMap template.FuncMap

type Template interface {
	ExecuteTemplate(wr io.Writer, name string, data interface{}) error
	Lookup(name string) *template.Template
	Templates() []*template.Template
	New(name string) *template.Template
	LoadTemplates(absPath string)
	LoadTemplatesWithPrefix(absPath, prefix string)
	AddTemplate(name, tpl string) error
	AddInternalTemplate(prefix, name, tpl string) error
	AddInternalShortcode(name, tpl string) error
	PrintErrors()
}

type templateErr struct {
	name string
	err  error
}

type GoHTMLTemplate struct {
	template.Template
	errors []*templateErr
}

// The "Global" Template System
func T() Template {
	if tmpl == nil {
		tmpl = New()
	}

	return tmpl
}

// Resets the internal template state to it's initial state
func InitializeT() Template {
	tmpl = New()
	return tmpl
}

// Return a new Hugo Template System
// With all the additional features, templates & functions
func New() Template {
	var templates = &GoHTMLTemplate{
		Template: *template.New(""),
		errors:   make([]*templateErr, 0),
	}

	localTemplates = &templates.Template

	templates.Funcs(funcMap)
	templates.LoadEmbedded()
	return templates
}

func Eq(x, y interface{}) bool {
	normalize := func(v interface{}) interface{} {
		vv := reflect.ValueOf(v)
		switch vv.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return vv.Int()
		case reflect.Float32, reflect.Float64:
			return vv.Float()
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return vv.Uint()
		default:
			return v
		}
	}
	x = normalize(x)
	y = normalize(y)
	return reflect.DeepEqual(x, y)
}

func Ne(x, y interface{}) bool {
	return !Eq(x, y)
}

func Ge(a, b interface{}) bool {
	left, right := compareGetFloat(a, b)
	return left >= right
}

func Gt(a, b interface{}) bool {
	left, right := compareGetFloat(a, b)
	return left > right
}

func Le(a, b interface{}) bool {
	left, right := compareGetFloat(a, b)
	return left <= right
}

func Lt(a, b interface{}) bool {
	left, right := compareGetFloat(a, b)
	return left < right
}

func compareGetFloat(a interface{}, b interface{}) (float64, float64) {
	var left, right float64
	var leftStr, rightStr *string
	var err error
	av := reflect.ValueOf(a)

	switch av.Kind() {
	case reflect.Array, reflect.Chan, reflect.Map, reflect.Slice:
		left = float64(av.Len())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		left = float64(av.Int())
	case reflect.Float32, reflect.Float64:
		left = av.Float()
	case reflect.String:
		left, err = strconv.ParseFloat(av.String(), 64)
		if err != nil {
			str := av.String()
			leftStr = &str
		}
	}

	bv := reflect.ValueOf(b)

	switch bv.Kind() {
	case reflect.Array, reflect.Chan, reflect.Map, reflect.Slice:
		right = float64(bv.Len())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		right = float64(bv.Int())
	case reflect.Float32, reflect.Float64:
		right = bv.Float()
	case reflect.String:
		right, err = strconv.ParseFloat(bv.String(), 64)
		if err != nil {
			str := bv.String()
			rightStr = &str
		}

	}

	switch {
	case leftStr == nil || rightStr == nil:
	case *leftStr < *rightStr:
		return 0, 1
	case *leftStr > *rightStr:
		return 1, 0
	default:
		return 0, 0
	}

	return left, right
}

func Substr(a interface{}, pos, length int) (string, error) {
	aStr, err := cast.ToStringE(a)
	if err != nil {
		return "", err
	}
	return aStr[pos:length], nil
}

func Split(a interface{}, delimiter string) ([]string, error) {
	aStr, err := cast.ToStringE(a)
	if err != nil {
		return []string{}, err
	}
	return strings.Split(aStr, delimiter), nil
}

func Intersect(l1, l2 interface{}) (interface{}, error) {
	if l1 == nil || l2 == nil {
		return make([]interface{}, 0), nil
	}

	l1v := reflect.ValueOf(l1)
	l2v := reflect.ValueOf(l2)

	switch l1v.Kind() {
	case reflect.Array, reflect.Slice:
		switch l2v.Kind() {
		case reflect.Array, reflect.Slice:
			r := reflect.MakeSlice(l1v.Type(), 0, 0)
			for i := 0; i < l1v.Len(); i++ {
				l1vv := l1v.Index(i)
				for j := 0; j < l2v.Len(); j++ {
					l2vv := l2v.Index(j)
					switch l1vv.Kind() {
					case reflect.String:
						if l1vv.Type() == l2vv.Type() && l1vv.String() == l2vv.String() && !In(r, l2vv) {
							r = reflect.Append(r, l2vv)
						}
					case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
						switch l2vv.Kind() {
						case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
							if l1vv.Int() == l2vv.Int() && !In(r, l2vv) {
								r = reflect.Append(r, l2vv)
							}
						}
					case reflect.Float32, reflect.Float64:
						switch l2vv.Kind() {
						case reflect.Float32, reflect.Float64:
							if l1vv.Float() == l2vv.Float() && !In(r, l2vv) {
								r = reflect.Append(r, l2vv)
							}
						}
					}
				}
			}
			return r.Interface(), nil
		default:
			return nil, errors.New("can't iterate over " + reflect.ValueOf(l2).Type().String())
		}
	default:
		return nil, errors.New("can't iterate over " + reflect.ValueOf(l1).Type().String())
	}
}

func In(l interface{}, v interface{}) bool {
	lv := reflect.ValueOf(l)
	vv := reflect.ValueOf(v)

	switch lv.Kind() {
	case reflect.Array, reflect.Slice:
		for i := 0; i < lv.Len(); i++ {
			lvv := lv.Index(i)
			switch lvv.Kind() {
			case reflect.String:
				if vv.Type() == lvv.Type() && vv.String() == lvv.String() {
					return true
				}
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				switch vv.Kind() {
				case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
					if vv.Int() == lvv.Int() {
						return true
					}
				}
			case reflect.Float32, reflect.Float64:
				switch vv.Kind() {
				case reflect.Float32, reflect.Float64:
					if vv.Float() == lvv.Float() {
						return true
					}
				}
			}
		}
	case reflect.String:
		if vv.Type() == lv.Type() && strings.Contains(lv.String(), vv.String()) {
			return true
		}
	}
	return false
}

// indirect is taken from 'text/template/exec.go'
func indirect(v reflect.Value) (rv reflect.Value, isNil bool) {
	for ; v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface; v = v.Elem() {
		if v.IsNil() {
			return v, true
		}
		if v.Kind() == reflect.Interface && v.NumMethod() > 0 {
			break
		}
	}
	return v, false
}

// First is exposed to templates, to iterate over the first N items in a
// rangeable list.
func First(limit interface{}, seq interface{}) (interface{}, error) {

	if limit == nil || seq == nil {
		return nil, errors.New("both limit and seq must be provided")
	}

	limitv, err := cast.ToIntE(limit)

	if err != nil {
		return nil, err
	}

	if limitv < 1 {
		return nil, errors.New("can't return negative/empty count of items from sequence")
	}

	seqv := reflect.ValueOf(seq)
	seqv, isNil := indirect(seqv)
	if isNil {
		return nil, errors.New("can't iterate over a nil value")
	}

	switch seqv.Kind() {
	case reflect.Array, reflect.Slice, reflect.String:
		// okay
	default:
		return nil, errors.New("can't iterate over " + reflect.ValueOf(seq).Type().String())
	}
	if limitv > seqv.Len() {
		limitv = seqv.Len()
	}
	return seqv.Slice(0, limitv).Interface(), nil
}

var (
	zero      reflect.Value
	errorType = reflect.TypeOf((*error)(nil)).Elem()
)

func evaluateSubElem(obj reflect.Value, elemName string) (reflect.Value, error) {
	if !obj.IsValid() {
		return zero, errors.New("can't evaluate an invalid value")
	}
	typ := obj.Type()
	obj, isNil := indirect(obj)

	// first, check whether obj has a method. In this case, obj is
	// an interface, a struct or its pointer. If obj is a struct,
	// to check all T and *T method, use obj pointer type Value
	objPtr := obj
	if objPtr.Kind() != reflect.Interface && objPtr.CanAddr() {
		objPtr = objPtr.Addr()
	}
	mt, ok := objPtr.Type().MethodByName(elemName)
	if ok {
		if mt.PkgPath != "" {
			return zero, fmt.Errorf("%s is an unexported method of type %s", elemName, typ)
		}
		// struct pointer has one receiver argument and interface doesn't have an argument
		if mt.Type.NumIn() > 1 || mt.Type.NumOut() == 0 || mt.Type.NumOut() > 2 {
			return zero, fmt.Errorf("%s is a method of type %s but doesn't satisfy requirements", elemName, typ)
		}
		if mt.Type.NumOut() == 1 && mt.Type.Out(0).Implements(errorType) {
			return zero, fmt.Errorf("%s is a method of type %s but doesn't satisfy requirements", elemName, typ)
		}
		if mt.Type.NumOut() == 2 && !mt.Type.Out(1).Implements(errorType) {
			return zero, fmt.Errorf("%s is a method of type %s but doesn't satisfy requirements", elemName, typ)
		}
		res := objPtr.Method(mt.Index).Call([]reflect.Value{})
		if len(res) == 2 && !res[1].IsNil() {
			return zero, fmt.Errorf("error at calling a method %s of type %s: %s", elemName, typ, res[1].Interface().(error))
		}
		return res[0], nil
	}

	// elemName isn't a method so next start to check whether it is
	// a struct field or a map value. In both cases, it mustn't be
	// a nil value
	if isNil {
		return zero, fmt.Errorf("can't evaluate a nil pointer of type %s by a struct field or map key name %s", typ, elemName)
	}
	switch obj.Kind() {
	case reflect.Struct:
		ft, ok := obj.Type().FieldByName(elemName)
		if ok {
			if ft.PkgPath != "" {
				return zero, fmt.Errorf("%s is an unexported field of struct type %s", elemName, typ)
			}
			return obj.FieldByIndex(ft.Index), nil
		}
		return zero, fmt.Errorf("%s isn't a field of struct type %s", elemName, typ)
	case reflect.Map:
		kv := reflect.ValueOf(elemName)
		if kv.Type().AssignableTo(obj.Type().Key()) {
			return obj.MapIndex(kv), nil
		}
		return zero, fmt.Errorf("%s isn't a key of map type %s", elemName, typ)
	}
	return zero, fmt.Errorf("%s is neither a struct field, a method nor a map element of type %s", elemName, typ)
}

func checkCondition(v, mv reflect.Value, op string) (bool, error) {
	if !v.IsValid() || !mv.IsValid() {
		return false, nil
	}

	var isNil bool
	v, isNil = indirect(v)
	if isNil {
		return false, nil
	}
	mv, isNil = indirect(mv)
	if isNil {
		return false, nil
	}

	var ivp, imvp *int64
	var svp, smvp *string
	var ima []int64
	var sma []string
	if mv.Type() == v.Type() {
		switch v.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			iv := v.Int()
			ivp = &iv
			imv := mv.Int()
			imvp = &imv
		case reflect.String:
			sv := v.String()
			svp = &sv
			smv := mv.String()
			smvp = &smv
		}
	} else {
		if mv.Kind() != reflect.Array && mv.Kind() != reflect.Slice {
			return false, nil
		}
		if mv.Type().Elem() != v.Type() {
			return false, nil
		}
		switch v.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			iv := v.Int()
			ivp = &iv
			for i := 0; i < mv.Len(); i++ {
				ima = append(ima, mv.Index(i).Int())
			}
		case reflect.String:
			sv := v.String()
			svp = &sv
			for i := 0; i < mv.Len(); i++ {
				sma = append(sma, mv.Index(i).String())
			}
		}
	}

	switch op {
	case "", "=", "==", "eq":
		if ivp != nil && imvp != nil {
			return *ivp == *imvp, nil
		} else if svp != nil && smvp != nil {
			return *svp == *smvp, nil
		}
	case "!=", "<>", "ne":
		if ivp != nil && imvp != nil {
			return *ivp != *imvp, nil
		} else if svp != nil && smvp != nil {
			return *svp != *smvp, nil
		}
	case ">=", "ge":
		if ivp != nil && imvp != nil {
			return *ivp >= *imvp, nil
		} else if svp != nil && smvp != nil {
			return *svp >= *smvp, nil
		}
	case ">", "gt":
		if ivp != nil && imvp != nil {
			return *ivp > *imvp, nil
		} else if svp != nil && smvp != nil {
			return *svp > *smvp, nil
		}
	case "<=", "le":
		if ivp != nil && imvp != nil {
			return *ivp <= *imvp, nil
		} else if svp != nil && smvp != nil {
			return *svp <= *smvp, nil
		}
	case "<", "lt":
		if ivp != nil && imvp != nil {
			return *ivp < *imvp, nil
		} else if svp != nil && smvp != nil {
			return *svp < *smvp, nil
		}
	case "in", "not in":
		var r bool
		if ivp != nil && len(ima) > 0 {
			r = In(ima, *ivp)
		} else if svp != nil {
			if len(sma) > 0 {
				r = In(sma, *svp)
			} else if smvp != nil {
				r = In(*smvp, *svp)
			}
		} else {
			return false, nil
		}
		if op == "not in" {
			return !r, nil
		} else {
			return r, nil
		}
	default:
		return false, errors.New("no such an operator")
	}
	return false, nil
}

func Where(seq, key interface{}, args ...interface{}) (r interface{}, err error) {
	seqv := reflect.ValueOf(seq)
	kv := reflect.ValueOf(key)

	var mv reflect.Value
	var op string
	switch len(args) {
	case 1:
		mv = reflect.ValueOf(args[0])
	case 2:
		var ok bool
		if op, ok = args[0].(string); !ok {
			return nil, errors.New("operator argument must be string type")
		}
		op = strings.TrimSpace(strings.ToLower(op))
		mv = reflect.ValueOf(args[1])
	default:
		return nil, errors.New("can't evaluate the array by no match argument or more than or equal to two arguments")
	}

	seqv, isNil := indirect(seqv)
	if isNil {
		return nil, errors.New("can't iterate over a nil value of type " + reflect.ValueOf(seq).Type().String())
	}

	var path []string
	if kv.Kind() == reflect.String {
		path = strings.Split(strings.Trim(kv.String(), "."), ".")
	}

	switch seqv.Kind() {
	case reflect.Array, reflect.Slice:
		rv := reflect.MakeSlice(seqv.Type(), 0, 0)
		for i := 0; i < seqv.Len(); i++ {
			var vvv reflect.Value
			rvv := seqv.Index(i)
			if kv.Kind() == reflect.String {
				vvv = rvv
				for _, elemName := range path {
					vvv, err = evaluateSubElem(vvv, elemName)
					if err != nil {
						return nil, err
					}
				}
			} else {
				vv, _ := indirect(rvv)
				if vv.Kind() == reflect.Map && kv.Type().AssignableTo(vv.Type().Key()) {
					vvv = vv.MapIndex(kv)
				}
			}
			if ok, err := checkCondition(vvv, mv, op); ok {
				rv = reflect.Append(rv, rvv)
			} else if err != nil {
				return nil, err
			}
		}
		return rv.Interface(), nil
	default:
		return nil, errors.New("can't iterate over " + reflect.ValueOf(seq).Type().String())
	}
}

// Apply, given a map, array, or slice, returns a new slice with the function fname applied over it.
func Apply(seq interface{}, fname string, args ...interface{}) (interface{}, error) {
	if seq == nil {
		return make([]interface{}, 0), nil
	}

	if fname == "apply" {
		return nil, errors.New("can't apply myself (no turtles allowed)")
	}

	seqv := reflect.ValueOf(seq)
	seqv, isNil := indirect(seqv)
	if isNil {
		return nil, errors.New("can't iterate over a nil value")
	}

	fn, found := funcMap[fname]
	if !found {
		return nil, errors.New("can't find function " + fname)
	}

	fnv := reflect.ValueOf(fn)

	switch seqv.Kind() {
	case reflect.Array, reflect.Slice:
		r := make([]interface{}, seqv.Len())
		for i := 0; i < seqv.Len(); i++ {
			vv := seqv.Index(i)

			vvv, err := applyFnToThis(fnv, vv, args...)

			if err != nil {
				return nil, err
			}

			r[i] = vvv.Interface()
		}

		return r, nil
	default:
		return nil, errors.New("can't apply over " + reflect.ValueOf(seq).Type().String())
	}
}

func applyFnToThis(fn, this reflect.Value, args ...interface{}) (reflect.Value, error) {
	n := make([]reflect.Value, len(args))
	for i, arg := range args {
		if arg == "." {
			n[i] = this
		} else {
			n[i] = reflect.ValueOf(arg)
		}
	}

	res := fn.Call(n)

	if len(res) == 1 || res[1].IsNil() {
		return res[0], nil
	} else {
		return reflect.ValueOf(nil), res[1].Interface().(error)
	}
}

func Delimit(seq, delimiter interface{}, last ...interface{}) (template.HTML, error) {
	d, err := cast.ToStringE(delimiter)
	if err != nil {
		return "", err
	}

	var dLast *string
	for _, l := range last {
		dStr, err := cast.ToStringE(l)
		if err != nil {
			dLast = nil
		}
		dLast = &dStr
		break
	}

	seqv := reflect.ValueOf(seq)
	seqv, isNil := indirect(seqv)
	if isNil {
		return "", errors.New("can't iterate over a nil value")
	}

	var str string
	switch seqv.Kind() {
	case reflect.Map:
		sortSeq, err := Sort(seq)
		if err != nil {
			return "", err
		}
		seqv = reflect.ValueOf(sortSeq)
		fallthrough
	case reflect.Array, reflect.Slice, reflect.String:
		for i := 0; i < seqv.Len(); i++ {
			val := seqv.Index(i).Interface()
			valStr, err := cast.ToStringE(val)
			if err != nil {
				continue
			}
			switch {
			case i == seqv.Len()-2 && dLast != nil:
				str += valStr + *dLast
			case i == seqv.Len()-1:
				str += valStr
			default:
				str += valStr + d
			}
		}

	default:
		return "", errors.New("can't iterate over " + reflect.ValueOf(seq).Type().String())
	}

	return template.HTML(str), nil
}

func Sort(seq interface{}, args ...interface{}) ([]interface{}, error) {
	seqv := reflect.ValueOf(seq)
	seqv, isNil := indirect(seqv)
	if isNil {
		return nil, errors.New("can't iterate over a nil value")
	}

	// Create a list of pairs that will be used to do the sort
	p := pairList{SortAsc: true}
	p.Pairs = make([]pair, seqv.Len())

	for i, l := range args {
		dStr, err := cast.ToStringE(l)
		switch {
		case i == 0 && err != nil:
			p.SortByField = ""
		case i == 0 && err == nil:
			p.SortByField = dStr
		case i == 1 && err == nil && dStr == "desc":
			p.SortAsc = false
		case i == 1:
			p.SortAsc = true
		}
	}

	var sorted []interface{}
	switch seqv.Kind() {
	case reflect.Array, reflect.Slice:
		for i := 0; i < seqv.Len(); i++ {
			p.Pairs[i].Key = reflect.ValueOf(i)
			p.Pairs[i].Value = seqv.Index(i)
		}
		if p.SortByField == "" {
			p.SortByField = "value"
		}

	case reflect.Map:
		keys := seqv.MapKeys()
		for i := 0; i < seqv.Len(); i++ {
			p.Pairs[i].Key = keys[i]
			p.Pairs[i].Value = seqv.MapIndex(keys[i])
		}

	default:
		return nil, errors.New("can't sort " + reflect.ValueOf(seq).Type().String())
	}
	sorted = p.sort()
	return sorted, nil
}

// Credit for pair sorting method goes to Andrew Gerrand
// https://groups.google.com/forum/#!topic/golang-nuts/FT7cjmcL7gw
// A data structure to hold a key/value pair.
type pair struct {
	Key   reflect.Value
	Value reflect.Value
}

// A slice of pairs that implements sort.Interface to sort by Value.
type pairList struct {
	Pairs       []pair
	SortByField string
	SortAsc     bool
}

func (p pairList) Swap(i, j int) { p.Pairs[i], p.Pairs[j] = p.Pairs[j], p.Pairs[i] }
func (p pairList) Len() int      { return len(p.Pairs) }
func (p pairList) Less(i, j int) bool {
	var truth bool
	switch {
	case p.SortByField == "value":
		iVal := p.Pairs[i].Value
		jVal := p.Pairs[j].Value
		truth = Lt(iVal.Interface(), jVal.Interface())

	case p.SortByField != "":
		if p.Pairs[i].Value.FieldByName(p.SortByField).IsValid() {
			iVal := p.Pairs[i].Value.FieldByName(p.SortByField)
			jVal := p.Pairs[j].Value.FieldByName(p.SortByField)
			truth = Lt(iVal.Interface(), jVal.Interface())
		}
	default:
		iVal := p.Pairs[i].Key
		jVal := p.Pairs[j].Key
		truth = Lt(iVal.Interface(), jVal.Interface())
	}
	return truth
}

// sorts a pairList and returns a slice of sorted values
func (p pairList) sort() []interface{} {
	if p.SortAsc {
		sort.Sort(p)
	} else {
		sort.Sort(sort.Reverse(p))
	}
	sorted := make([]interface{}, len(p.Pairs))
	for i, v := range p.Pairs {
		sorted[i] = v.Value.Interface()
	}

	return sorted
}

func IsSet(a interface{}, key interface{}) bool {
	av := reflect.ValueOf(a)
	kv := reflect.ValueOf(key)

	switch av.Kind() {
	case reflect.Array, reflect.Chan, reflect.Slice:
		if int64(av.Len()) > kv.Int() {
			return true
		}
	case reflect.Map:
		if kv.Type() == av.Type().Key() {
			return av.MapIndex(kv).IsValid()
		}
	}

	return false
}

func ReturnWhenSet(a, k interface{}) interface{} {
	av, isNil := indirect(reflect.ValueOf(a))
	if isNil {
		return ""
	}

	var avv reflect.Value
	switch av.Kind() {
	case reflect.Array, reflect.Slice:
		index, ok := k.(int)
		if ok && av.Len() > index {
			avv = av.Index(index)
		}
	case reflect.Map:
		kv := reflect.ValueOf(k)
		if kv.Type().AssignableTo(av.Type().Key()) {
			avv = av.MapIndex(kv)
		}
	}

	if avv.IsValid() {
		switch avv.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return avv.Int()
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return avv.Uint()
		case reflect.Float32, reflect.Float64:
			return avv.Float()
		case reflect.String:
			return avv.String()
		}
	}

	return ""
}

func Highlight(in interface{}, lang string) template.HTML {
	var str string
	av := reflect.ValueOf(in)
	switch av.Kind() {
	case reflect.String:
		str = av.String()
	}

	return template.HTML(helpers.Highlight(html.UnescapeString(str), lang))
}

func Markdownify(text string) template.HTML {
	return template.HTML(helpers.RenderBytes(&helpers.RenderingContext{Content: []byte(text), PageFmt: "markdown"}))
}

func refPage(page interface{}, ref, methodName string) template.HTML {
	value := reflect.ValueOf(page)

	method := value.MethodByName(methodName)

	if method.IsValid() && method.Type().NumIn() == 1 && method.Type().NumOut() == 2 {
		result := method.Call([]reflect.Value{reflect.ValueOf(ref)})

		url, err := result[0], result[1]

		if !err.IsNil() {
			jww.ERROR.Printf("%s", err.Interface())
			return template.HTML(fmt.Sprintf("%s", err.Interface()))
		}

		if url.String() == "" {
			jww.ERROR.Printf("ref %s could not be found\n", ref)
			return template.HTML(ref)
		}

		return template.HTML(url.String())
	}

	jww.ERROR.Printf("Can only create references from Page and Node objects.")
	return template.HTML(ref)
}

func Ref(page interface{}, ref string) template.HTML {
	return refPage(page, ref, "Ref")
}

func RelRef(page interface{}, ref string) template.HTML {
	return refPage(page, ref, "RelRef")
}

func Chomp(text interface{}) (string, error) {
	s, err := cast.ToStringE(text)
	if err != nil {
		return "", err
	}

	return strings.TrimRight(s, "\r\n"), nil
}

// Trim leading/trailing characters defined by b from a
func Trim(a interface{}, b string) (string, error) {
	aStr, err := cast.ToStringE(a)
	if err != nil {
		return "", err
	}
	return strings.Trim(aStr, b), nil
}

// Replace all occurences of b with c in a
func Replace(a, b, c interface{}) (string, error) {
	aStr, err := cast.ToStringE(a)
	if err != nil {
		return "", err
	}
	bStr, err := cast.ToStringE(b)
	if err != nil {
		return "", err
	}
	cStr, err := cast.ToStringE(c)
	if err != nil {
		return "", err
	}
	return strings.Replace(aStr, bStr, cStr, -1), nil
}

// DateFormat converts the textual representation of the datetime string into
// the other form or returns it of the time.Time value. These are formatted
// with the layout string
func DateFormat(layout string, v interface{}) (string, error) {
	t, err := cast.ToTimeE(v)
	if err != nil {
		return "", err
	}
	return t.Format(layout), nil
}

func SafeHTML(text string) template.HTML {
	return template.HTML(text)
}

// "safeHTMLAttr" is currently disabled, pending further discussion
// on its use case.  2015-01-19
func SafeHTMLAttr(text string) template.HTMLAttr {
	return template.HTMLAttr(text)
}

func SafeCSS(text string) template.CSS {
	return template.CSS(text)
}

func SafeURL(text string) template.URL {
	return template.URL(text)
}

func doArithmetic(a, b interface{}, op rune) (interface{}, error) {
	av := reflect.ValueOf(a)
	bv := reflect.ValueOf(b)
	var ai, bi int64
	var af, bf float64
	var au, bu uint64
	switch av.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		ai = av.Int()
		switch bv.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			bi = bv.Int()
		case reflect.Float32, reflect.Float64:
			af = float64(ai) // may overflow
			ai = 0
			bf = bv.Float()
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			bu = bv.Uint()
			if ai >= 0 {
				au = uint64(ai)
				ai = 0
			} else {
				bi = int64(bu) // may overflow
				bu = 0
			}
		default:
			return nil, errors.New("Can't apply the operator to the values")
		}
	case reflect.Float32, reflect.Float64:
		af = av.Float()
		switch bv.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			bf = float64(bv.Int()) // may overflow
		case reflect.Float32, reflect.Float64:
			bf = bv.Float()
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			bf = float64(bv.Uint()) // may overflow
		default:
			return nil, errors.New("Can't apply the operator to the values")
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		au = av.Uint()
		switch bv.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			bi = bv.Int()
			if bi >= 0 {
				bu = uint64(bi)
				bi = 0
			} else {
				ai = int64(au) // may overflow
				au = 0
			}
		case reflect.Float32, reflect.Float64:
			af = float64(au) // may overflow
			au = 0
			bf = bv.Float()
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			bu = bv.Uint()
		default:
			return nil, errors.New("Can't apply the operator to the values")
		}
	case reflect.String:
		as := av.String()
		if bv.Kind() == reflect.String && op == '+' {
			bs := bv.String()
			return as + bs, nil
		} else {
			return nil, errors.New("Can't apply the operator to the values")
		}
	default:
		return nil, errors.New("Can't apply the operator to the values")
	}

	switch op {
	case '+':
		if ai != 0 || bi != 0 {
			return ai + bi, nil
		} else if af != 0 || bf != 0 {
			return af + bf, nil
		} else if au != 0 || bu != 0 {
			return au + bu, nil
		} else {
			return 0, nil
		}
	case '-':
		if ai != 0 || bi != 0 {
			return ai - bi, nil
		} else if af != 0 || bf != 0 {
			return af - bf, nil
		} else if au != 0 || bu != 0 {
			return au - bu, nil
		} else {
			return 0, nil
		}
	case '*':
		if ai != 0 || bi != 0 {
			return ai * bi, nil
		} else if af != 0 || bf != 0 {
			return af * bf, nil
		} else if au != 0 || bu != 0 {
			return au * bu, nil
		} else {
			return 0, nil
		}
	case '/':
		if bi != 0 {
			return ai / bi, nil
		} else if bf != 0 {
			return af / bf, nil
		} else if bu != 0 {
			return au / bu, nil
		} else {
			return nil, errors.New("Can't divide the value by 0")
		}
	default:
		return nil, errors.New("There is no such an operation")
	}
}

func Mod(a, b interface{}) (int64, error) {
	av := reflect.ValueOf(a)
	bv := reflect.ValueOf(b)
	var ai, bi int64

	switch av.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		ai = av.Int()
	default:
		return 0, errors.New("Modulo operator can't be used with non integer value")
	}

	switch bv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		bi = bv.Int()
	default:
		return 0, errors.New("Modulo operator can't be used with non integer value")
	}

	if bi == 0 {
		return 0, errors.New("The number can't be divided by zero at modulo operation")
	}

	return ai % bi, nil
}

func ModBool(a, b interface{}) (bool, error) {
	res, err := Mod(a, b)
	if err != nil {
		return false, err
	}
	return res == int64(0), nil
}

func Partial(name string, context_list ...interface{}) template.HTML {
	if strings.HasPrefix("partials/", name) {
		name = name[8:]
	}
	var context interface{}

	if len(context_list) == 0 {
		context = nil
	} else {
		context = context_list[0]
	}
	return ExecuteTemplateToHTML(context, "partials/"+name, "theme/partials/"+name)
}

func ExecuteTemplate(context interface{}, buffer *bytes.Buffer, layouts ...string) {
	worked := false
	for _, layout := range layouts {

		name := layout

		if localTemplates.Lookup(name) == nil {
			name = layout + ".html"
		}

		if localTemplates.Lookup(name) != nil {
			err := localTemplates.ExecuteTemplate(buffer, name, context)
			if err != nil {
				jww.ERROR.Println(err, "in", name)
			}
			worked = true
			break
		}
	}
	if !worked {
		jww.ERROR.Println("Unable to render", layouts)
		jww.ERROR.Println("Expecting to find a template in either the theme/layouts or /layouts in one of the following relative locations", layouts)
	}
}

func ExecuteTemplateToHTML(context interface{}, layouts ...string) template.HTML {
	b := bp.GetBuffer()
	defer bp.PutBuffer(b)
	ExecuteTemplate(context, b, layouts...)
	return template.HTML(b.String())
}

func (t *GoHTMLTemplate) LoadEmbedded() {
	t.EmbedShortcodes()
	t.EmbedTemplates()
}

func (t *GoHTMLTemplate) AddInternalTemplate(prefix, name, tpl string) error {
	if prefix != "" {
		return t.AddTemplate("_internal/"+prefix+"/"+name, tpl)
	} else {
		return t.AddTemplate("_internal/"+name, tpl)
	}
}

func (t *GoHTMLTemplate) AddInternalShortcode(name, content string) error {
	return t.AddInternalTemplate("shortcodes", name, content)
}

func (t *GoHTMLTemplate) AddTemplate(name, tpl string) error {
	_, err := t.New(name).Parse(tpl)
	if err != nil {
		t.errors = append(t.errors, &templateErr{name: name, err: err})
	}
	return err
}

func (t *GoHTMLTemplate) AddTemplateFile(name, path string) error {
	// get the suffix and switch on that
	ext := filepath.Ext(path)
	switch ext {
	case ".amber":
		compiler := amber.New()
		// Parse the input file
		if err := compiler.ParseFile(path); err != nil {
			return nil
		}

		if _, err := compiler.CompileWithTemplate(t.New(name)); err != nil {
			return err
		}
	case ".ace":
		b, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}
		name = name[:len(name)-len(ext)] + ".html"
		base := ace.NewFile(path, b)
		inner := ace.NewFile("", []byte{})
		rslt, err := ace.ParseSource(ace.NewSource(base, inner, []*ace.File{}), nil)
		if err != nil {
			t.errors = append(t.errors, &templateErr{name: name, err: err})
			return err
		}
		_, err = ace.CompileResultWithTemplate(t.New(name), rslt, nil)
		if err != nil {
			t.errors = append(t.errors, &templateErr{name: name, err: err})
		}
		return err
	default:
		b, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}

		return t.AddTemplate(name, string(b))
	}

	return nil

}

func (t *GoHTMLTemplate) GenerateTemplateNameFrom(base, path string) string {
	name, _ := filepath.Rel(base, path)
	return filepath.ToSlash(name)
}

func isDotFile(path string) bool {
	return filepath.Base(path)[0] == '.'
}

func isBackupFile(path string) bool {
	return path[len(path)-1] == '~'
}

func (t *GoHTMLTemplate) loadTemplates(absPath string, prefix string) {
	walker := func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if fi.Mode()&os.ModeSymlink == os.ModeSymlink {
			link, err := filepath.EvalSymlinks(absPath)
			if err != nil {
				jww.ERROR.Printf("Cannot read symbolic link '%s', error was: %s", absPath, err)
				return nil
			}
			linkfi, err := os.Stat(link)
			if err != nil {
				jww.ERROR.Printf("Cannot stat '%s', error was: %s", link, err)
				return nil
			}
			if !linkfi.Mode().IsRegular() {
				jww.ERROR.Printf("Symbolic links for directories not supported, skipping '%s'", absPath)
			}
			return nil
		}

		if !fi.IsDir() {
			if isDotFile(path) || isBackupFile(path) {
				return nil
			}

			tplName := t.GenerateTemplateNameFrom(absPath, path)

			if prefix != "" {
				tplName = strings.Trim(prefix, "/") + "/" + tplName
			}

			t.AddTemplateFile(tplName, path)

		}
		return nil
	}

	filepath.Walk(absPath, walker)
}

func (t *GoHTMLTemplate) LoadTemplatesWithPrefix(absPath string, prefix string) {
	t.loadTemplates(absPath, prefix)
}

func (t *GoHTMLTemplate) LoadTemplates(absPath string) {
	t.loadTemplates(absPath, "")
}

func (t *GoHTMLTemplate) PrintErrors() {
	for _, e := range t.errors {
		jww.ERROR.Println(e.err)
	}
}

func init() {
	funcMap = template.FuncMap{
		"urlize":      helpers.URLize,
		"sanitizeURL": helpers.SanitizeURL,
		"sanitizeurl": helpers.SanitizeURL,
		"eq":          Eq,
		"ne":          Ne,
		"gt":          Gt,
		"ge":          Ge,
		"lt":          Lt,
		"le":          Le,
		"in":          In,
		"substr":      Substr,
		"split":       Split,
		"intersect":   Intersect,
		"isSet":       IsSet,
		"isset":       IsSet,
		"echoParam":   ReturnWhenSet,
		"safeHTML":    SafeHTML,
		"safeCSS":     SafeCSS,
		"safeURL":     SafeURL,
		"markdownify": Markdownify,
		"first":       First,
		"where":       Where,
		"delimit":     Delimit,
		"sort":        Sort,
		"highlight":   Highlight,
		"add":         func(a, b interface{}) (interface{}, error) { return doArithmetic(a, b, '+') },
		"sub":         func(a, b interface{}) (interface{}, error) { return doArithmetic(a, b, '-') },
		"div":         func(a, b interface{}) (interface{}, error) { return doArithmetic(a, b, '/') },
		"mod":         Mod,
		"mul":         func(a, b interface{}) (interface{}, error) { return doArithmetic(a, b, '*') },
		"modBool":     ModBool,
		"lower":       func(a string) string { return strings.ToLower(a) },
		"upper":       func(a string) string { return strings.ToUpper(a) },
		"title":       func(a string) string { return strings.Title(a) },
		"partial":     Partial,
		"ref":         Ref,
		"relref":      RelRef,
		"apply":       Apply,
		"chomp":       Chomp,
		"replace":     Replace,
		"trim":        Trim,
		"dateFormat":  DateFormat,
		"getJSON":     GetJSON,
		"getCSV":      GetCSV,
		"seq":         helpers.Seq,
		"getenv":      func(varName string) string { return os.Getenv(varName) },

		// "getJson" is deprecated. Will be removed in 0.15.
		"getJson": func(urlParts ...string) interface{} {
			helpers.Deprecated("Template", "getJson", "getJSON")
			return GetJSON(urlParts...)
		},
		// "getJson" is deprecated. Will be removed in 0.15.
		"getCsv": func(sep string, urlParts ...string) [][]string {
			helpers.Deprecated("Template", "getCsv", "getCSV")
			return GetCSV(sep, urlParts...)
		},
		// "safeHtml" is deprecated. Will be removed in 0.15.
		"safeHtml": func(text string) template.HTML {
			helpers.Deprecated("Template", "safeHtml", "safeHTML")
			return SafeHTML(text)
		},
		// "safeCss" is deprecated. Will be removed in 0.15.
		"safeCss": func(text string) template.CSS {
			helpers.Deprecated("Template", "safeCss", "safeCSS")
			return SafeCSS(text)
		},
		// "safeUrl" is deprecated. Will be removed in 0.15.
		"safeUrl": func(text string) template.URL {
			helpers.Deprecated("Template", "safeUrl", "safeURL")
			return SafeURL(text)
		},
	}

}
