package util

import (
	"bytes"
	"fmt"
	"io"
	"reflect"
	"strings"
	"text/tabwriter"
)

const (
	NotAvailable = "-"
)

type Preparer interface {
	Prepare()
}

func Table(in interface{}) []byte {
	if p, ok := in.(Preparer); ok {
		p.Prepare()
	}
	out, err := JsonTab(in, 0, 0, 2, ' ', tabwriter.TabIndent)
	if err != nil {
		return []byte(err.Error())
	}
	return out
}

func TableStr(in interface{}) string {
	out, err := JsonTabStr(in, 0, 0, 2, ' ', tabwriter.TabIndent)
	if err != nil {
		return err.Error()
	}
	return out
}

func jsonTagStruct(w io.Writer, rv reflect.Value) {
	rt := rv.Type()
	for i := 0; i < rt.NumField(); i++ {
		fv := rv.Field(i)
		ff := rt.Field(i)

		if !fv.CanInterface() {
			continue
		}

		if tag, ok := ff.Tag.Lookup("json"); ok && strings.HasSuffix(tag, ",inline") {
			if fv.Kind() == reflect.Ptr {
				if fv.IsNil() {
					continue
				}
				fv = fv.Elem()
			}
			if fv.Kind() == reflect.Struct {
				jsonTagStruct(w, fv)
			}
			continue
		}

		out := ff.Tag.Get("out")
		if out == "-" {
			continue
		}

		if outs := strings.Split(out, ","); len(outs) > 0 && outs[0] != "" {
			fmt.Fprintf(w, "%s\t", outs[0])
		} else {
			fmt.Fprintf(w, "%s\t", ff.Name)
		}

		fmt.Fprintf(w, "%s\n", fieldOut(fv.Interface(), out))
	}
}

func jsonTabArrayTitle(w io.Writer, v reflect.Value) {
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		fieldv := v.Field(i)
		tag := field.Tag

		if !fieldv.CanInterface() {
			continue
		}

		out := tag.Get("out")
		if out == "-" {
			continue
		}

		jsonTags := strings.Split(tag.Get("json"), ",")
		inline := false
		if len(jsonTags) > 1 {
			if StringArrayContains("inline", jsonTags[1:]) {
				inline = true
			}
		}
		if inline {
			if field.Type.Kind() == reflect.Struct {
				jsonTabArrayTitle(w, fieldv)
			}
			continue
		}

		if i > 0 {
			fmt.Fprintf(w, "\t")
		}

		name := field.Name
		outs := strings.Split(out, ",")
		if len(outs) > 0 && outs[0] != "" {
			name = outs[0]
		}

		fmt.Fprintf(w, "%s", name)
	}
}

func jsonTabArrayEntity(w io.Writer, rv reflect.Value) {
	rt := rv.Type()

	for i := 0; i < rv.NumField(); i++ {
		fv := rv.Field(i)
		ff := rt.Field(i)
		tag := ff.Tag

		if !fv.CanInterface() {
			continue
		}

		out := tag.Get("out")
		if out == "-" {
			continue
		}

		jsonTags := strings.Split(tag.Get("json"), ",")
		inline := false
		if len(jsonTags) > 1 {
			if StringArrayContains("inline", jsonTags[1:]) {
				inline = true
			}
		}
		if inline {
			if ff.Type.Kind() == reflect.Struct {
				jsonTabArrayEntity(w, fv)
			}
			continue
		}

		if i > 0 {
			fmt.Fprintf(w, "\t")
		}

		if n, _ := fmt.Fprintf(w, "%v", fieldOut(fv.Interface(), out)); n == 0 {
			fmt.Fprintf(w, NotAvailable)
		}
	}

}

func jsonTabArray(w io.Writer, v reflect.Value) {
	for i := 0; i < v.Len(); i++ {
		fieldv := reflect.Indirect(v.Index(i))
		if i == 0 {
			jsonTabArrayTitle(w, fieldv)
			fmt.Fprintf(w, "\n")
		}
		jsonTabArrayEntity(w, fieldv)
		fmt.Fprintf(w, "\n")
	}
}

func PrettyTab(in string) string {
	buf := bytes.NewBuffer([]byte{})
	w := tabwriter.NewWriter(buf, 0, 0, 2, ' ', tabwriter.TabIndent)
	fmt.Fprintf(w, in)
	w.Flush()
	return buf.String()
}

func Printable(in interface{}) bool {
	switch reflect.Indirect(reflect.ValueOf(in)).Kind() {
	case reflect.Struct, reflect.Slice, reflect.Array, reflect.Map:
		return true
	default:
		return false
	}
}

// JsonTabStr(f, 0, 0, 2, ' ', tabwriter.TabIndent)
func JsonTab(in interface{}, minwidth, tabwidth, padding int, padchar byte, flags uint) ([]byte, error) {
	v := reflect.Indirect(reflect.ValueOf(in))

	out := bytes.NewBuffer([]byte{})
	w := tabwriter.NewWriter(out, minwidth, tabwidth, padding, padchar, flags)

	switch v.Kind() {
	case reflect.Struct:
		jsonTagStruct(w, v)
	case reflect.Slice, reflect.Array:
		jsonTabArray(w, v)
	case reflect.Map:
		for _, key := range v.MapKeys() {
			fmt.Fprintf(w, "%s\t%v\n",
				key, v.MapIndex(key).Interface())
		}
	default:
		return []byte{}, fmt.Errorf("unsupported type: %s", v.Type())
	}

	w.Flush()
	return out.Bytes(), nil
}

func JsonTabStr(in interface{}, minwidth, tabwidth, padding int, padchar byte, flags uint) (string, error) {
	b, err := JsonTab(in, minwidth, tabwidth, padding, padchar, flags)
	return string(b), err
}

func MustJsonTabStr(in interface{}, minwidth, tabwidth, padding int, padchar byte, flags uint) string {
	out, err := JsonTabStr(in, minwidth, tabwidth, padding, padchar, flags)
	if err != nil {
		panic(err)
	}
	return out
}

func fieldOut(v interface{}, out string) string {
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return NotAvailable
		}
		rv = rv.Elem()
	}

	v = rv.Interface()

	args := strings.Split(out, ",")

	if len(args) < 2 {
		return fmt.Sprintf("%v", v)
	}

	switch args[1] {
	case "fromNow":
		if data, ok := v.(int64); ok {
			if data == 0 {
				return NotAvailable
			}
			return FromNow(data)
		}
	case "date":
		if data, ok := v.(int64); ok {
			if data == 0 {
				return NotAvailable
			}
			return FmtTs(data)
		}
	case "oneline":
		if data, ok := v.(string); ok {
			return FirstLine(data)
		}
	case "substr":
		if data, ok := v.(string); ok {
			return SubStr2(data, Atoi(args[2]), Atoi(args[3]))
		}
	}
	return fmt.Sprintf("%v", v)
}
