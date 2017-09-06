package tsm1_test

import (
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/influxdata/influxdb/tsdb/engine/tsm1"
)

func TestTSMReader_Type(t *testing.T) {
	dir := MustTempDir()
	defer os.RemoveAll(dir)
	f := MustTempFile(dir)

	w, err := tsm1.NewTSMWriter(f)
	if err != nil {
		t.Fatalf("unexpected error creating writer: %v", err)
	}

	values := []tsm1.Value{tsm1.NewValue(0, int64(1))}
	if err := w.Write([]byte("cpu"), values); err != nil {
		t.Fatalf("unexpected error writing: %v", err)

	}
	if err := w.WriteIndex(); err != nil {
		t.Fatalf("unexpected error closing: %v", err)
	}

	if err := w.Close(); err != nil {
		t.Fatalf("unexpected error closing: %v", err)
	}

	f, err = os.Open(f.Name())
	if err != nil {
		t.Fatalf("unexpected error opening: %v", err)
	}
	r, err := tsm1.NewTSMReader(f)
	if err != nil {
		t.Fatalf("unexpected error created reader: %v", err)
	}

	typ, err := r.Type([]byte("cpu"))
	if err != nil {
		fatal(t, "reading type", err)
	}

	if got, exp := typ, tsm1.BlockInteger; got != exp {
		t.Fatalf("type mismatch: got %v, exp %v", got, exp)
	}
}

func TestTSMReader_MMAP_ReadAll(t *testing.T) {
	dir := MustTempDir()
	defer os.RemoveAll(dir)
	f := MustTempFile(dir)
	defer f.Close()

	w, err := tsm1.NewTSMWriter(f)
	if err != nil {
		t.Fatalf("unexpected error creating writer: %v", err)
	}

	var data = map[string][]tsm1.Value{
		"float":  []tsm1.Value{tsm1.NewValue(1, 1.0)},
		"int":    []tsm1.Value{tsm1.NewValue(1, int64(1))},
		"uint":   []tsm1.Value{tsm1.NewValue(1, ^uint64(0))},
		"bool":   []tsm1.Value{tsm1.NewValue(1, true)},
		"string": []tsm1.Value{tsm1.NewValue(1, "foo")},
	}

	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		if err := w.Write([]byte(k), data[k]); err != nil {
			t.Fatalf("unexpected error writing: %v", err)
		}
	}

	if err := w.WriteIndex(); err != nil {
		t.Fatalf("unexpected error writing index: %v", err)
	}

	if err := w.Close(); err != nil {
		t.Fatalf("unexpected error closing: %v", err)
	}

	f, err = os.Open(f.Name())
	if err != nil {
		t.Fatalf("unexpected error open file: %v", err)
	}

	r, err := tsm1.NewTSMReader(f)
	if err != nil {
		t.Fatalf("unexpected error created reader: %v", err)
	}
	defer r.Close()

	var count int
	for k, vals := range data {
		readValues, err := r.ReadAll([]byte(k))
		if err != nil {
			t.Fatalf("unexpected error readin: %v", err)
		}

		if exp := len(vals); exp != len(readValues) {
			t.Fatalf("read values length mismatch: got %v, exp %v", len(readValues), exp)
		}

		for i, v := range vals {
			if v.Value() != readValues[i].Value() {
				t.Fatalf("read value mismatch(%d): got %v, exp %d", i, readValues[i].Value(), v.Value())
			}
		}
		count++
	}

	if got, exp := count, len(data); got != exp {
		t.Fatalf("read values count mismatch: got %v, exp %v", got, exp)
	}
}

func TestTSMReader_MMAP_Read(t *testing.T) {
	dir := MustTempDir()
	defer os.RemoveAll(dir)
	f := MustTempFile(dir)
	defer f.Close()

	w, err := tsm1.NewTSMWriter(f)
	if err != nil {
		t.Fatalf("unexpected error creating writer: %v", err)
	}

	var data = map[string][]tsm1.Value{
		"float": []tsm1.Value{
			tsm1.NewValue(1, 1.0)},
		"int": []tsm1.Value{
			tsm1.NewValue(1, int64(1))},
		"uint": []tsm1.Value{
			tsm1.NewValue(1, ^uint64(0))},
		"bool": []tsm1.Value{
			tsm1.NewValue(1, true)},
		"string": []tsm1.Value{
			tsm1.NewValue(1, "foo")},
	}

	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		if err := w.Write([]byte(k), data[k]); err != nil {
			t.Fatalf("unexpected error writing: %v", err)
		}
	}

	if err := w.WriteIndex(); err != nil {
		t.Fatalf("unexpected error writing index: %v", err)
	}

	if err := w.Close(); err != nil {
		t.Fatalf("unexpected error closing: %v", err)
	}

	f, err = os.Open(f.Name())
	if err != nil {
		t.Fatalf("unexpected error open file: %v", err)
	}

	r, err := tsm1.NewTSMReader(f)
	if err != nil {
		t.Fatalf("unexpected error created reader: %v", err)
	}
	defer r.Close()

	var count int
	for k, vals := range data {
		readValues, err := r.Read([]byte(k), vals[0].UnixNano())
		if err != nil {
			t.Fatalf("unexpected error readin: %v", err)
		}

		if exp := len(vals); exp != len(readValues) {
			t.Fatalf("read values length mismatch: got %v, exp %v", len(readValues), exp)
		}

		for i, v := range vals {
			if v.Value() != readValues[i].Value() {
				t.Fatalf("read value mismatch(%d): got %v, exp %d", i, readValues[i].Value(), v.Value())
			}
		}
		count++
	}

	if got, exp := count, len(data); got != exp {
		t.Fatalf("read values count mismatch: got %v, exp %v", got, exp)
	}
}

func TestTSMReader_MMAP_Keys(t *testing.T) {
	dir := MustTempDir()
	defer os.RemoveAll(dir)
	f := MustTempFile(dir)
	defer f.Close()

	w, err := tsm1.NewTSMWriter(f)
	if err != nil {
		t.Fatalf("unexpected error creating writer: %v", err)
	}

	var data = map[string][]tsm1.Value{
		"float": []tsm1.Value{
			tsm1.NewValue(1, 1.0)},
		"int": []tsm1.Value{
			tsm1.NewValue(1, int64(1))},
		"uint": []tsm1.Value{
			tsm1.NewValue(1, ^uint64(0))},
		"bool": []tsm1.Value{
			tsm1.NewValue(1, true)},
		"string": []tsm1.Value{
			tsm1.NewValue(1, "foo")},
	}

	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		if err := w.Write([]byte(k), data[k]); err != nil {
			t.Fatalf("unexpected error writing: %v", err)
		}
	}

	if err := w.WriteIndex(); err != nil {
		t.Fatalf("unexpected error writing index: %v", err)
	}

	if err := w.Close(); err != nil {
		t.Fatalf("unexpected error closing: %v", err)
	}

	f, err = os.Open(f.Name())
	if err != nil {
		t.Fatalf("unexpected error open file: %v", err)
	}

	r, err := tsm1.NewTSMReader(f)
	if err != nil {
		t.Fatalf("unexpected error created reader: %v", err)
	}
	defer r.Close()

	var count int
	for k, vals := range data {
		readValues, err := r.Read([]byte(k), vals[0].UnixNano())
		if err != nil {
			t.Fatalf("unexpected error readin: %v", err)
		}

		if exp := len(vals); exp != len(readValues) {
			t.Fatalf("read values length mismatch: got %v, exp %v", len(readValues), exp)
		}

		for i, v := range vals {
			if v.Value() != readValues[i].Value() {
				t.Fatalf("read value mismatch(%d): got %v, exp %d", i, readValues[i].Value(), v.Value())
			}
		}
		count++
	}

	if got, exp := count, len(data); got != exp {
		t.Fatalf("read values count mismatch: got %v, exp %v", got, exp)
	}
}

func TestTSMReader_MMAP_Tombstone(t *testing.T) {
	dir := MustTempDir()
	defer os.RemoveAll(dir)
	f := MustTempFile(dir)
	defer f.Close()

	w, err := tsm1.NewTSMWriter(f)
	if err != nil {
		t.Fatalf("unexpected error creating writer: %v", err)
	}

	values := []tsm1.Value{tsm1.NewValue(0, 1.0)}
	if err := w.Write([]byte("cpu"), values); err != nil {
		t.Fatalf("unexpected error writing: %v", err)
	}

	if err := w.Write([]byte("mem"), values); err != nil {
		t.Fatalf("unexpected error writing: %v", err)
	}

	if err := w.WriteIndex(); err != nil {
		t.Fatalf("unexpected error writing index: %v", err)
	}

	if err := w.Close(); err != nil {
		t.Fatalf("unexpected error closing: %v", err)
	}

	f, err = os.Open(f.Name())
	if err != nil {
		t.Fatalf("unexpected error open file: %v", err)
	}

	r, err := tsm1.NewTSMReader(f)
	if err != nil {
		t.Fatalf("unexpected error created reader: %v", err)
	}

	if err := r.Delete([][]byte{[]byte("mem")}); err != nil {
		t.Fatalf("unexpected error deleting: %v", err)
	}

	r, err = tsm1.NewTSMReader(f)
	if err != nil {
		t.Fatalf("unexpected error created reader: %v", err)
	}
	defer r.Close()

	if got, exp := r.KeyCount(), 1; got != exp {
		t.Fatalf("key length mismatch: got %v, exp %v", got, exp)
	}
}

func TestTSMReader_MMAP_TombstoneRange(t *testing.T) {
	dir := MustTempDir()
	defer os.RemoveAll(dir)
	f := MustTempFile(dir)
	defer f.Close()

	w, err := tsm1.NewTSMWriter(f)
	if err != nil {
		t.Fatalf("unexpected error creating writer: %v", err)
	}

	expValues := []tsm1.Value{
		tsm1.NewValue(1, 1.0),
		tsm1.NewValue(2, 2.0),
		tsm1.NewValue(3, 3.0),
	}
	if err := w.Write([]byte("cpu"), expValues); err != nil {
		t.Fatalf("unexpected error writing: %v", err)
	}

	if err := w.WriteIndex(); err != nil {
		t.Fatalf("unexpected error writing index: %v", err)
	}

	if err := w.Close(); err != nil {
		t.Fatalf("unexpected error closing: %v", err)
	}

	f, err = os.Open(f.Name())
	if err != nil {
		t.Fatalf("unexpected error open file: %v", err)
	}

	r, err := tsm1.NewTSMReader(f)
	if err != nil {
		t.Fatalf("unexpected error created reader: %v", err)
	}

	if err := r.DeleteRange([][]byte{[]byte("cpu")}, 2, math.MaxInt64); err != nil {
		t.Fatalf("unexpected error deleting: %v", err)
	}
	defer r.Close()

	if got, exp := r.ContainsValue([]byte("cpu"), 1), true; got != exp {
		t.Fatalf("ContainsValue mismatch: got %v, exp %v", got, exp)
	}

	if got, exp := r.ContainsValue([]byte("cpu"), 3), false; got != exp {
		t.Fatalf("ContainsValue mismatch: got %v, exp %v", got, exp)
	}

	values, err := r.ReadAll([]byte("cpu"))
	if err != nil {
		t.Fatalf("unexpected error reading all: %v", err)
	}

	if got, exp := len(values), 1; got != exp {
		t.Fatalf("values length mismatch: got %v, exp %v", got, exp)
	}

	if got, exp := values[0].String(), expValues[0].String(); got != exp {
		t.Fatalf("value mismatch: got %v, exp %v", got, exp)
	}
}

func TestTSMReader_MMAP_TombstoneOutsideTimeRange(t *testing.T) {
	dir := MustTempDir()
	defer os.RemoveAll(dir)
	f := MustTempFile(dir)
	defer f.Close()

	w, err := tsm1.NewTSMWriter(f)
	if err != nil {
		t.Fatalf("unexpected error creating writer: %v", err)
	}

	expValues := []tsm1.Value{
		tsm1.NewValue(1, 1.0),
		tsm1.NewValue(2, 2.0),
		tsm1.NewValue(3, 3.0),
	}
	if err := w.Write([]byte("cpu"), expValues); err != nil {
		t.Fatalf("unexpected error writing: %v", err)
	}

	if err := w.WriteIndex(); err != nil {
		t.Fatalf("unexpected error writing index: %v", err)
	}

	if err := w.Close(); err != nil {
		t.Fatalf("unexpected error closing: %v", err)
	}

	f, err = os.Open(f.Name())
	if err != nil {
		t.Fatalf("unexpected error open file: %v", err)
	}

	r, err := tsm1.NewTSMReader(f)
	if err != nil {
		t.Fatalf("unexpected error created reader: %v", err)
	}

	if err := r.DeleteRange([][]byte{[]byte("cpu")}, 0, 0); err != nil {
		t.Fatalf("unexpected error deleting: %v", err)
	}
	defer r.Close()

	if got, exp := r.ContainsValue([]byte("cpu"), 1), true; got != exp {
		t.Fatalf("ContainsValue mismatch: got %v, exp %v", got, exp)
	}

	if got, exp := r.ContainsValue([]byte("cpu"), 2), true; got != exp {
		t.Fatalf("ContainsValue mismatch: got %v, exp %v", got, exp)
	}

	if got, exp := r.ContainsValue([]byte("cpu"), 3), true; got != exp {
		t.Fatalf("ContainsValue mismatch: got %v, exp %v", got, exp)
	}

	if got, exp := r.HasTombstones(), false; got != exp {
		t.Fatalf("HasTombstones mismatch: got %v, exp %v", got, exp)
	}

	if got, exp := len(r.TombstoneFiles()), 0; got != exp {
		t.Fatalf("TombstoneFiles len mismatch: got %v, exp %v", got, exp)
	}
}

func TestTSMReader_MMAP_TombstoneOutsideKeyRange(t *testing.T) {
	dir := MustTempDir()
	defer os.RemoveAll(dir)
	f := MustTempFile(dir)
	defer f.Close()

	w, err := tsm1.NewTSMWriter(f)
	if err != nil {
		t.Fatalf("unexpected error creating writer: %v", err)
	}

	expValues := []tsm1.Value{
		tsm1.NewValue(1, 1.0),
		tsm1.NewValue(2, 2.0),
		tsm1.NewValue(3, 3.0),
	}
	if err := w.Write([]byte("cpu"), expValues); err != nil {
		t.Fatalf("unexpected error writing: %v", err)
	}

	if err := w.WriteIndex(); err != nil {
		t.Fatalf("unexpected error writing index: %v", err)
	}

	if err := w.Close(); err != nil {
		t.Fatalf("unexpected error closing: %v", err)
	}

	f, err = os.Open(f.Name())
	if err != nil {
		t.Fatalf("unexpected error open file: %v", err)
	}

	r, err := tsm1.NewTSMReader(f)
	if err != nil {
		t.Fatalf("unexpected error created reader: %v", err)
	}

	if err := r.DeleteRange([][]byte{[]byte("mem")}, 0, 3); err != nil {
		t.Fatalf("unexpected error deleting: %v", err)
	}
	defer r.Close()

	if got, exp := r.ContainsValue([]byte("cpu"), 1), true; got != exp {
		t.Fatalf("ContainsValue mismatch: got %v, exp %v", got, exp)
	}

	if got, exp := r.ContainsValue([]byte("cpu"), 2), true; got != exp {
		t.Fatalf("ContainsValue mismatch: got %v, exp %v", got, exp)
	}

	if got, exp := r.ContainsValue([]byte("cpu"), 3), true; got != exp {
		t.Fatalf("ContainsValue mismatch: got %v, exp %v", got, exp)
	}

	if got, exp := r.HasTombstones(), false; got != exp {
		t.Fatalf("HasTombstones mismatch: got %v, exp %v", got, exp)
	}

	if got, exp := len(r.TombstoneFiles()), 0; got != exp {
		t.Fatalf("TombstoneFiles len mismatch: got %v, exp %v", got, exp)

	}
}

func TestTSMReader_MMAP_TombstoneOverlapKeyRange(t *testing.T) {
	dir := MustTempDir()
	defer os.RemoveAll(dir)
	f := MustTempFile(dir)
	defer f.Close()

	w, err := tsm1.NewTSMWriter(f)
	if err != nil {
		t.Fatalf("unexpected error creating writer: %v", err)
	}

	expValues := []tsm1.Value{
		tsm1.NewValue(1, 1.0),
		tsm1.NewValue(2, 2.0),
		tsm1.NewValue(3, 3.0),
	}
	if err := w.Write([]byte("cpu,app=foo,host=server-0#!~#value"), expValues); err != nil {
		t.Fatalf("unexpected error writing: %v", err)
	}

	if err := w.Write([]byte("cpu,app=foo,host=server-73379#!~#value"), expValues); err != nil {
		t.Fatalf("unexpected error writing: %v", err)
	}

	if err := w.WriteIndex(); err != nil {
		t.Fatalf("unexpected error writing index: %v", err)
	}

	if err := w.Close(); err != nil {
		t.Fatalf("unexpected error closing: %v", err)
	}

	f, err = os.Open(f.Name())
	if err != nil {
		t.Fatalf("unexpected error open file: %v", err)
	}

	r, err := tsm1.NewTSMReader(f)
	if err != nil {
		t.Fatalf("unexpected error created reader: %v", err)
	}

	if err := r.DeleteRange([][]byte{
		[]byte("cpu,app=foo,host=server-0#!~#value"),
		[]byte("cpu,app=foo,host=server-73379#!~#value"),
		[]byte("cpu,app=foo,host=server-99999#!~#value")},
		math.MinInt64, math.MaxInt64); err != nil {
		t.Fatalf("unexpected error deleting: %v", err)
	}
	defer r.Close()

	if got, exp := r.Contains([]byte("cpu,app=foo,host=server-0#!~#value")), false; got != exp {
		t.Fatalf("ContainsValue mismatch: got %v, exp %v", got, exp)
	}

	if got, exp := r.Contains([]byte("cpu,app=foo,host=server-73379#!~#value")), false; got != exp {
		t.Fatalf("ContainsValue mismatch: got %v, exp %v", got, exp)
	}

	if got, exp := r.HasTombstones(), true; got != exp {
		t.Fatalf("HasTombstones mismatch: got %v, exp %v", got, exp)
	}

	if got, exp := len(r.TombstoneFiles()), 1; got != exp {
		t.Fatalf("TombstoneFiles len mismatch: got %v, exp %v", got, exp)
	}
}

func TestTSMReader_MMAP_TombstoneFullRange(t *testing.T) {
	dir := MustTempDir()
	defer os.RemoveAll(dir)
	f := MustTempFile(dir)
	defer f.Close()

	w, err := tsm1.NewTSMWriter(f)
	if err != nil {
		t.Fatalf("unexpected error creating writer: %v", err)
	}

	expValues := []tsm1.Value{
		tsm1.NewValue(1, 1.0),
		tsm1.NewValue(2, 2.0),
		tsm1.NewValue(3, 3.0),
	}
	if err := w.Write([]byte("cpu"), expValues); err != nil {
		t.Fatalf("unexpected error writing: %v", err)
	}

	if err := w.WriteIndex(); err != nil {
		t.Fatalf("unexpected error writing index: %v", err)
	}

	if err := w.Close(); err != nil {
		t.Fatalf("unexpected error closing: %v", err)
	}

	f, err = os.Open(f.Name())
	if err != nil {
		t.Fatalf("unexpected error open file: %v", err)
	}

	r, err := tsm1.NewTSMReader(f)
	if err != nil {
		t.Fatalf("unexpected error created reader: %v", err)
	}

	if err := r.DeleteRange([][]byte{[]byte("cpu")}, math.MinInt64, math.MaxInt64); err != nil {
		t.Fatalf("unexpected error deleting: %v", err)
	}
	defer r.Close()

	values, err := r.ReadAll([]byte("cpu"))
	if err != nil {
		t.Fatalf("unexpected error reading all: %v", err)
	}

	if got, exp := len(values), 0; got != exp {
		t.Fatalf("values length mismatch: got %v, exp %v", got, exp)
	}
}

func TestTSMReader_MMAP_TombstoneMultipleRanges(t *testing.T) {
	dir := MustTempDir()
	defer os.RemoveAll(dir)
	f := MustTempFile(dir)
	defer f.Close()

	w, err := tsm1.NewTSMWriter(f)
	if err != nil {
		t.Fatalf("unexpected error creating writer: %v", err)
	}

	expValues := []tsm1.Value{
		tsm1.NewValue(1, 1.0),
		tsm1.NewValue(2, 2.0),
		tsm1.NewValue(3, 3.0),
		tsm1.NewValue(4, 4.0),
		tsm1.NewValue(5, 5.0),
	}
	if err := w.Write([]byte("cpu"), expValues); err != nil {
		t.Fatalf("unexpected error writing: %v", err)
	}

	if err := w.WriteIndex(); err != nil {
		t.Fatalf("unexpected error writing index: %v", err)
	}

	if err := w.Close(); err != nil {
		t.Fatalf("unexpected error closing: %v", err)
	}

	f, err = os.Open(f.Name())
	if err != nil {
		t.Fatalf("unexpected error open file: %v", err)
	}

	r, err := tsm1.NewTSMReader(f)
	if err != nil {
		t.Fatalf("unexpected error created reader: %v", err)
	}
	defer r.Close()

	if err := r.DeleteRange([][]byte{[]byte("cpu")}, 2, 2); err != nil {
		t.Fatalf("unexpected error deleting: %v", err)
	}

	if err := r.DeleteRange([][]byte{[]byte("cpu")}, 4, 4); err != nil {
		t.Fatalf("unexpected error deleting: %v", err)
	}

	values, err := r.ReadAll([]byte("cpu"))
	if err != nil {
		t.Fatalf("unexpected error reading all: %v", err)
	}

	if got, exp := len(values), 3; got != exp {
		t.Fatalf("values length mismatch: got %v, exp %v", got, exp)
	}
}

func TestTSMReader_MMAP_TombstoneOutsideRange(t *testing.T) {
	dir := MustTempDir()
	defer os.RemoveAll(dir)
	f := MustTempFile(dir)
	defer f.Close()

	w, err := tsm1.NewTSMWriter(f)
	if err != nil {
		t.Fatalf("unexpected error creating writer: %v", err)
	}

	cpuValues := []tsm1.Value{
		tsm1.NewValue(1, 1.0),
		tsm1.NewValue(2, 2.0),
		tsm1.NewValue(3, 3.0),
	}
	if err := w.Write([]byte("cpu"), cpuValues); err != nil {
		t.Fatalf("unexpected error writing: %v", err)
	}

	memValues := []tsm1.Value{
		tsm1.NewValue(1, 1.0),
		tsm1.NewValue(2, 2.0),
		tsm1.NewValue(30, 3.0),
	}
	if err := w.Write([]byte("mem"), memValues); err != nil {
		t.Fatalf("unexpected error writing: %v", err)
	}

	if err := w.WriteIndex(); err != nil {
		t.Fatalf("unexpected error writing index: %v", err)
	}

	if err := w.Close(); err != nil {
		t.Fatalf("unexpected error closing: %v", err)
	}

	f, err = os.Open(f.Name())
	if err != nil {
		t.Fatalf("unexpected error open file: %v", err)
	}

	r, err := tsm1.NewTSMReader(f)
	if err != nil {
		t.Fatalf("unexpected error created reader: %v", err)
	}

	if err := r.DeleteRange([][]byte{[]byte("cpu"), []byte("mem")}, 5, math.MaxInt64); err != nil {
		t.Fatalf("unexpected error deleting: %v", err)
	}
	defer r.Close()

	if got, exp := r.KeyCount(), 2; got != exp {
		t.Fatalf("key count mismatch: got %v, exp %v", got, exp)
	}

	if got, exp := len(r.TombstoneRange([]byte("cpu"))), 0; got != exp {
		t.Fatalf("tombstone range mismatch: got %v, exp %v", got, exp)
	}

	values, err := r.ReadAll([]byte("cpu"))
	if err != nil {
		t.Fatalf("unexpected error reading all: %v", err)
	}

	if got, exp := len(values), len(cpuValues); got != exp {
		t.Fatalf("values length mismatch: got %v, exp %v", got, exp)
	}

	if got, exp := len(r.TombstoneRange([]byte("mem"))), 1; got != exp {
		t.Fatalf("tombstone range mismatch: got %v, exp %v", got, exp)
	}

	values, err = r.ReadAll([]byte("mem"))
	if err != nil {
		t.Fatalf("unexpected error reading all: %v", err)
	}

	if got, exp := len(values), len(memValues[:2]); got != exp {
		t.Fatalf("values length mismatch: got %v, exp %v", got, exp)
	}

}

func TestTSMReader_MMAP_Stats(t *testing.T) {
	dir := MustTempDir()
	defer os.RemoveAll(dir)
	f := MustTempFile(dir)
	defer f.Close()

	w, err := tsm1.NewTSMWriter(f)
	if err != nil {
		t.Fatalf("unexpected error creating writer: %v", err)
	}

	values1 := []tsm1.Value{tsm1.NewValue(0, 1.0)}
	if err := w.Write([]byte("cpu"), values1); err != nil {
		t.Fatalf("unexpected error writing: %v", err)
	}

	values2 := []tsm1.Value{tsm1.NewValue(1, 1.0)}
	if err := w.Write([]byte("mem"), values2); err != nil {
		t.Fatalf("unexpected error writing: %v", err)
	}

	if err := w.WriteIndex(); err != nil {
		t.Fatalf("unexpected error writing index: %v", err)
	}

	if err := w.Close(); err != nil {
		t.Fatalf("unexpected error closing: %v", err)
	}

	f, err = os.Open(f.Name())
	if err != nil {
		t.Fatalf("unexpected error open file: %v", err)
	}

	r, err := tsm1.NewTSMReader(f)
	if err != nil {
		t.Fatalf("unexpected error created reader: %v", err)
	}
	defer r.Close()

	stats := r.Stats()
	if got, exp := string(stats.MinKey), "cpu"; got != exp {
		t.Fatalf("min key mismatch: got %v, exp %v", got, exp)
	}

	if got, exp := string(stats.MaxKey), "mem"; got != exp {
		t.Fatalf("max key mismatch: got %v, exp %v", got, exp)
	}

	if got, exp := stats.MinTime, values1[0].UnixNano(); got != exp {
		t.Fatalf("min time mismatch: got %v, exp %v", got, exp)
	}

	if got, exp := stats.MaxTime, values2[0].UnixNano(); got != exp {
		t.Fatalf("max time mismatch: got %v, exp %v", got, exp)
	}

	if got, exp := r.KeyCount(), 2; got != exp {
		t.Fatalf("key length mismatch: got %v, exp %v", got, exp)
	}
}

// Ensure that we return an error if we try to open a non-tsm file
func TestTSMReader_VerifiesFileType(t *testing.T) {
	dir := MustTempDir()
	defer os.RemoveAll(dir)
	f := MustTempFile(dir)
	defer f.Close()

	// write some garbage
	f.Write([]byte{0x23, 0xac, 0x99, 0x22, 0x77, 0x23, 0xac, 0x99, 0x22, 0x77, 0x23, 0xac, 0x99, 0x22, 0x77, 0x23, 0xac, 0x99, 0x22, 0x77})

	_, err := tsm1.NewTSMReader(f)
	if err == nil {
		t.Fatal("expected error trying to open non-tsm file")
	}
}

func TestIndirectIndex_Entries(t *testing.T) {
	index := tsm1.NewIndexWriter()
	index.Add([]byte("cpu"), tsm1.BlockFloat64, 0, 1, 10, 100)
	index.Add([]byte("cpu"), tsm1.BlockFloat64, 2, 3, 20, 200)
	index.Add([]byte("mem"), tsm1.BlockFloat64, 0, 1, 10, 100)
	exp := index.Entries([]byte("cpu"))

	b, err := index.MarshalBinary()
	if err != nil {
		t.Fatalf("unexpected error marshaling index: %v", err)
	}

	indirect := tsm1.NewIndirectIndex()
	if err := indirect.UnmarshalBinary(b); err != nil {
		t.Fatalf("unexpected error unmarshaling index: %v", err)
	}

	entries := indirect.Entries([]byte("cpu"))

	if got, exp := len(entries), len(exp); got != exp {
		t.Fatalf("entries length mismatch: got %v, exp %v", got, exp)
	}

	for i, exp := range exp {
		got := entries[i]
		if exp.MinTime != got.MinTime {
			t.Fatalf("minTime mismatch: got %v, exp %v", got.MinTime, exp.MinTime)
		}

		if exp.MaxTime != got.MaxTime {
			t.Fatalf("minTime mismatch: got %v, exp %v", got.MaxTime, exp.MaxTime)
		}

		if exp.Size != got.Size {
			t.Fatalf("size mismatch: got %v, exp %v", got.Size, exp.Size)
		}
		if exp.Offset != got.Offset {
			t.Fatalf("size mismatch: got %v, exp %v", got.Offset, exp.Offset)
		}
	}
}

func TestIndirectIndex_Entries_NonExistent(t *testing.T) {
	index := tsm1.NewIndexWriter()
	index.Add([]byte("cpu"), tsm1.BlockFloat64, 0, 1, 10, 100)
	index.Add([]byte("cpu"), tsm1.BlockFloat64, 2, 3, 20, 200)

	b, err := index.MarshalBinary()
	if err != nil {
		t.Fatalf("unexpected error marshaling index: %v", err)
	}

	indirect := tsm1.NewIndirectIndex()
	if err := indirect.UnmarshalBinary(b); err != nil {
		t.Fatalf("unexpected error unmarshaling index: %v", err)
	}

	// mem has not been added to the index so we should get no entries back
	// for both
	exp := index.Entries([]byte("mem"))
	entries := indirect.Entries([]byte("mem"))

	if got, exp := len(entries), len(exp); got != exp && exp != 0 {
		t.Fatalf("entries length mismatch: got %v, exp %v", got, exp)
	}
}

func TestIndirectIndex_MaxBlocks(t *testing.T) {
	index := tsm1.NewIndexWriter()
	for i := 0; i < 1<<16; i++ {
		index.Add([]byte("cpu"), tsm1.BlockFloat64, 0, 1, 10, 20)
	}

	if _, err := index.MarshalBinary(); err == nil {
		t.Fatalf("expected max block count error. got nil")
	} else {
		println(err.Error())
	}
}

func TestIndirectIndex_Type(t *testing.T) {
	index := tsm1.NewIndexWriter()
	index.Add([]byte("cpu"), tsm1.BlockInteger, 0, 1, 10, 20)

	b, err := index.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}

	ind := tsm1.NewIndirectIndex()
	if err := ind.UnmarshalBinary(b); err != nil {
		fatal(t, "unmarshal binary", err)
	}

	typ, err := ind.Type([]byte("cpu"))
	if err != nil {
		fatal(t, "reading type", err)
	}

	if got, exp := typ, tsm1.BlockInteger; got != exp {
		t.Fatalf("type mismatch: got %v, exp %v", got, exp)
	}
}

func TestIndirectIndex_Keys(t *testing.T) {
	index := tsm1.NewIndexWriter()
	index.Add([]byte("cpu"), tsm1.BlockFloat64, 0, 1, 10, 20)
	index.Add([]byte("cpu"), tsm1.BlockFloat64, 1, 2, 20, 30)
	index.Add([]byte("mem"), tsm1.BlockFloat64, 0, 1, 10, 20)

	keys := index.Keys()

	// 2 distinct keys
	if got, exp := len(keys), 2; got != exp {
		t.Fatalf("length mismatch: got %v, exp %v", got, exp)
	}

	// Keys should be sorted
	if got, exp := string(keys[0]), "cpu"; got != exp {
		t.Fatalf("key mismatch: got %v, exp %v", got, exp)
	}

	if got, exp := string(keys[1]), "mem"; got != exp {
		t.Fatalf("key mismatch: got %v, exp %v", got, exp)
	}
}

func TestBlockIterator_Single(t *testing.T) {
	dir := MustTempDir()
	defer os.RemoveAll(dir)
	f := MustTempFile(dir)

	w, err := tsm1.NewTSMWriter(f)
	if err != nil {
		t.Fatalf("unexpected error creating writer: %v", err)
	}

	values := []tsm1.Value{tsm1.NewValue(0, int64(1))}
	if err := w.Write([]byte("cpu"), values); err != nil {
		t.Fatalf("unexpected error writing: %v", err)

	}
	if err := w.WriteIndex(); err != nil {
		t.Fatalf("unexpected error closing: %v", err)
	}

	if err := w.Close(); err != nil {
		t.Fatalf("unexpected error closing: %v", err)
	}

	fd, err := os.Open(f.Name())
	if err != nil {
		t.Fatalf("unexpected error opening: %v", err)
	}

	r, err := tsm1.NewTSMReader(fd)
	if err != nil {
		t.Fatalf("unexpected error created reader: %v", err)
	}

	var count int
	iter := r.BlockIterator()
	for iter.Next() {
		key, minTime, maxTime, typ, _, buf, err := iter.Read()
		if err != nil {
			t.Fatalf("unexpected error creating iterator: %v", err)
		}

		if got, exp := string(key), "cpu"; got != exp {
			t.Fatalf("key mismatch: got %v, exp %v", got, exp)
		}

		if got, exp := minTime, int64(0); got != exp {
			t.Fatalf("min time mismatch: got %v, exp %v", got, exp)
		}

		if got, exp := maxTime, int64(0); got != exp {
			t.Fatalf("max time mismatch: got %v, exp %v", got, exp)
		}

		if got, exp := typ, tsm1.BlockInteger; got != exp {
			t.Fatalf("block type mismatch: got %v, exp %v", got, exp)
		}

		if len(buf) == 0 {
			t.Fatalf("buf length = 0")
		}

		count++
	}

	if got, exp := count, len(values); got != exp {
		t.Fatalf("value count mismatch: got %v, exp %v", got, exp)
	}
}

func TestBlockIterator_MultipleBlocks(t *testing.T) {
	dir := MustTempDir()
	defer os.RemoveAll(dir)
	f := MustTempFile(dir)

	w, err := tsm1.NewTSMWriter(f)
	if err != nil {
		t.Fatalf("unexpected error creating writer: %v", err)
	}

	values1 := []tsm1.Value{tsm1.NewValue(0, int64(1))}
	if err := w.Write([]byte("cpu"), values1); err != nil {
		t.Fatalf("unexpected error writing: %v", err)
	}

	values2 := []tsm1.Value{tsm1.NewValue(1, int64(2))}
	if err := w.Write([]byte("cpu"), values2); err != nil {
		t.Fatalf("unexpected error writing: %v", err)
	}

	if err := w.WriteIndex(); err != nil {
		t.Fatalf("unexpected error closing: %v", err)
	}

	if err := w.Close(); err != nil {
		t.Fatalf("unexpected error closing: %v", err)
	}

	fd, err := os.Open(f.Name())
	if err != nil {
		t.Fatalf("unexpected error opening: %v", err)
	}

	r, err := tsm1.NewTSMReader(fd)
	if err != nil {
		t.Fatalf("unexpected error created reader: %v", err)
	}

	var count int
	expData := []tsm1.Values{values1, values2}
	iter := r.BlockIterator()
	var i int
	for iter.Next() {
		key, minTime, maxTime, typ, _, buf, err := iter.Read()

		if err != nil {
			t.Fatalf("unexpected error creating iterator: %v", err)
		}

		if got, exp := string(key), "cpu"; got != exp {
			t.Fatalf("key mismatch: got %v, exp %v", got, exp)
		}

		if got, exp := minTime, expData[i][0].UnixNano(); got != exp {
			t.Fatalf("min time mismatch: got %v, exp %v", got, exp)
		}

		if got, exp := maxTime, expData[i][0].UnixNano(); got != exp {
			t.Fatalf("max time mismatch: got %v, exp %v", got, exp)
		}

		if got, exp := typ, tsm1.BlockInteger; got != exp {
			t.Fatalf("block type mismatch: got %v, exp %v", got, exp)
		}

		if len(buf) == 0 {
			t.Fatalf("buf length = 0")
		}

		count++
		i++
	}

	if got, exp := count, 2; got != exp {
		t.Fatalf("value count mismatch: got %v, exp %v", got, exp)
	}
}

func TestBlockIterator_Sorted(t *testing.T) {
	dir := MustTempDir()
	defer os.RemoveAll(dir)
	f := MustTempFile(dir)

	w, err := tsm1.NewTSMWriter(f)
	if err != nil {
		t.Fatalf("unexpected error creating writer: %v", err)
	}

	values := map[string][]tsm1.Value{
		"mem":    []tsm1.Value{tsm1.NewValue(0, int64(1))},
		"cycles": []tsm1.Value{tsm1.NewValue(0, ^uint64(0))},
		"cpu":    []tsm1.Value{tsm1.NewValue(1, float64(2))},
		"disk":   []tsm1.Value{tsm1.NewValue(1, true)},
		"load":   []tsm1.Value{tsm1.NewValue(1, "string")},
	}

	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		if err := w.Write([]byte(k), values[k]); err != nil {
			t.Fatalf("unexpected error writing: %v", err)

		}
	}

	if err := w.WriteIndex(); err != nil {
		t.Fatalf("unexpected error closing: %v", err)
	}

	if err := w.Close(); err != nil {
		t.Fatalf("unexpected error closing: %v", err)
	}

	fd, err := os.Open(f.Name())
	if err != nil {
		t.Fatalf("unexpected error opening: %v", err)
	}

	r, err := tsm1.NewTSMReader(fd)
	if err != nil {
		t.Fatalf("unexpected error created reader: %v", err)
	}

	var count int
	iter := r.BlockIterator()
	var lastKey string
	for iter.Next() {
		key, _, _, _, _, buf, err := iter.Read()

		if string(key) < lastKey {
			t.Fatalf("keys not sorted: got %v, last %v", key, lastKey)
		}

		lastKey = string(key)

		if err != nil {
			t.Fatalf("unexpected error creating iterator: %v", err)
		}

		if len(buf) == 0 {
			t.Fatalf("buf length = 0")
		}

		count++
	}

	if got, exp := count, len(values); got != exp {
		t.Fatalf("value count mismatch: got %v, exp %v", got, exp)
	}
}

func TestIndirectIndex_UnmarshalBinary_BlockCountOverflow(t *testing.T) {
	dir := MustTempDir()
	defer os.RemoveAll(dir)
	f := MustTempFile(dir)
	defer f.Close()

	w, err := tsm1.NewTSMWriter(f)
	if err != nil {
		t.Fatalf("unexpected error creating writer: %v", err)
	}

	for i := 0; i < 3280; i++ {
		w.Write([]byte("cpu"), []tsm1.Value{tsm1.NewValue(int64(i), float64(i))})
	}

	if err := w.WriteIndex(); err != nil {
		t.Fatalf("unexpected error closing: %v", err)
	}

	if err := w.Close(); err != nil {
		t.Fatalf("unexpected error closing: %v", err)
	}

	f, err = os.Open(f.Name())
	if err != nil {
		t.Fatalf("unexpected error open file: %v", err)
	}

	r, err := tsm1.NewTSMReader(f)
	if err != nil {
		t.Fatalf("unexpected error created reader: %v", err)
	}
	defer r.Close()
}

func TestCompacted_NotFull(t *testing.T) {
	dir := MustTempDir()
	defer os.RemoveAll(dir)
	f := MustTempFile(dir)

	w, err := tsm1.NewTSMWriter(f)
	if err != nil {
		t.Fatalf("unexpected error creating writer: %v", err)
	}

	values := []tsm1.Value{tsm1.NewValue(0, 1.0)}
	if err := w.Write([]byte("cpu"), values); err != nil {
		t.Fatalf("unexpected error writing: %v", err)

	}
	if err := w.WriteIndex(); err != nil {
		t.Fatalf("unexpected error writing index: %v", err)
	}

	if err := w.Close(); err != nil {
		t.Fatalf("unexpected error closing: %v", err)
	}

	fd, err := os.Open(f.Name())
	if err != nil {
		t.Fatalf("unexpected error open file: %v", err)
	}

	r, err := tsm1.NewTSMReader(fd)
	if err != nil {
		t.Fatalf("unexpected error created reader: %v", err)
	}

	iter := r.BlockIterator()
	if !iter.Next() {
		t.Fatalf("expected next, got false")
	}

	_, _, _, _, _, block, err := iter.Read()
	if err != nil {
		t.Fatalf("unexpected error reading block: %v", err)
	}

	if got, exp := tsm1.BlockCount(block), 1; got != exp {
		t.Fatalf("block count mismatch: got %v, exp %v", got, exp)
	}
}

func TestTSMReader_File_ReadAll(t *testing.T) {
	dir := MustTempDir()
	defer os.RemoveAll(dir)
	f := MustTempFile(dir)
	defer f.Close()

	w, err := tsm1.NewTSMWriter(f)
	if err != nil {
		t.Fatalf("unexpected error creating writer: %v", err)
	}

	var data = map[string][]tsm1.Value{
		"float": []tsm1.Value{
			tsm1.NewValue(1, 1.0)},
		"int": []tsm1.Value{
			tsm1.NewValue(1, int64(1))},
		"uint": []tsm1.Value{
			tsm1.NewValue(1, ^uint64(0))},
		"bool": []tsm1.Value{
			tsm1.NewValue(1, true)},
		"string": []tsm1.Value{
			tsm1.NewValue(1, "foo")},
	}

	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		if err := w.Write([]byte(k), data[k]); err != nil {
			t.Fatalf("unexpected error writing: %v", err)
		}
	}

	if err := w.WriteIndex(); err != nil {
		t.Fatalf("unexpected error writing index: %v", err)
	}

	if err := w.Close(); err != nil {
		t.Fatalf("unexpected error closing: %v", err)
	}

	f, err = os.Open(f.Name())
	if err != nil {
		t.Fatalf("unexpected error open file: %v", err)
	}

	r, err := tsm1.NewTSMReader(f)
	if err != nil {
		t.Fatalf("unexpected error created reader: %v", err)
	}
	defer r.Close()

	var count int
	for k, vals := range data {
		readValues, err := r.ReadAll([]byte(k))
		if err != nil {
			t.Fatalf("unexpected error reading: %v", err)
		}

		if exp := len(vals); exp != len(readValues) {
			t.Fatalf("read values length mismatch: exp %v, got %v", exp, len(readValues))
		}

		for i, v := range vals {
			if exp, got := v.Value(), readValues[i].Value(); exp != got {
				t.Fatalf("read value mismatch(%d): exp %v, got %d", i, v.Value(), readValues[i].Value())
			}
		}
		count++
	}

	if exp, got := len(data), count; exp != got {
		t.Fatalf("read values count mismatch: exp %v, got %v", exp, got)
	}
}

func TestTSMReader_FuzzCrashes(t *testing.T) {
	cases := []string{
		"",
		"\x16\xd1\x16\xd1\x01\x10\x14X\xfb\x03\xac~\x80\xf0\x00\x00\x00I^K" +
			"_\xf0\x00\x00\x00D424259389w\xf0\x00\x00\x00" +
			"o\x93\bO\x10?\xf0\x00\x00\x00\x00\b\x00\xc2_\xff\xd8\x0fX^" +
			"/\xbf\xe8\x00\x00\x00\x00\x00\x01\x00\bctr#!~#n\x00" +
			"\x00\x01\x14X\xfb\xb0\x03\xac~\x80\x14X\xfb\xb1\x00\xd4ܥ\x00\x00" +
			"\x00\x00\x00\x00\x00\x05\x00\x00\x00@\x00\x00\x00\x00\x00\x00\x00E",
		"\x16\xd1\x16\xd1\x01\x80'Z\\\x00\v)\x00\x00\x00\x00;\x9a\xca\x00" +
			"\x01\x05\x10?\xf0\x00\x00\x00\x00\x00\x00\xc2_\xff\xd6\x1d\xd4&\xed\v" +
			"\xc5\xf7\xfb\xc0\x00\x00\x00\x00\x00 \x00\x06a#!~#v\x00\x00" +
			"\x01\x00\x00\x00\x00;\x9a\xca\x00\x00\x00\x00\x01*\x05\xf2\x00\x00\x00\x00" +
			"\x00\x00\x00\x00\x00\x00\x00\x00\x002",
		"\x16\xd1\x16\xd1\x01\x80\xf0\x00\x00\x00I^K_\xf0\x00\x00\x00D7" +
			"\nw\xf0\x00\x00\x00o\x93\bO\x10?\xf0\x00\x00\x00\x00\x00\x00\xc2" +
			"_\xff\x14X\xfb\xb0\x03\xac~\x80\x14X\xfb\xb1\x00\xd4ܥ\x00\x00" +
			"\x00\x00\x00\x00\x00\x05\x00\x00\x00@\x00\x00\x00\x00\x00\x00\x00E",
		"\x16\xd1\x16\xd1\x01000000000000000" +
			"00000000000000000000" +
			"0000000000\x00\x000\x00\x0100000" +
			"000\x00\x00\x00\x00\x00\x00\x002",
		"\x16\xd1\x16\xd1\x01",
		"\x16\xd1\x16\xd1\x01\x00\x00o\x93\bO\x10?\xf0\x00\x00\x00\x00X^" +
			"/\xbf\xe8\x00\x00\x00\x00\x00\x01\x00\bctr#!~#n\x00" +
			"\x00\x01\x14X\xfb\xb0\x03\xac~\x80\x14X\xfb\xb1\x00\xd4ܥ\x00\x00" +
			"\x00\x00\x00\x00\x00\x05\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00E",
	}

	for _, c := range cases {
		func() {
			dir := MustTempDir()
			defer os.RemoveAll(dir)

			filename := filepath.Join(dir, "x.tsm")
			if err := ioutil.WriteFile(filename, []byte(c), 0600); err != nil {
				t.Fatalf("exp no error, got %s", err)
			}
			defer os.RemoveAll(dir)

			f, err := os.Open(filename)
			if err != nil {
				t.Fatalf("exp no error, got %s", err)
			}
			defer f.Close()

			r, err := tsm1.NewTSMReader(f)
			if err != nil {
				return
			}
			defer r.Close()

			iter := r.BlockIterator()
			for iter.Next() {
				key, _, _, _, _, _, err := iter.Read()
				if err != nil {
					return
				}

				_, _ = r.Type(key)

				if _, err = r.ReadAll(key); err != nil {
					return
				}
			}
		}()
	}
}

func TestTSMReader_File_Read(t *testing.T) {
	dir := MustTempDir()
	defer os.RemoveAll(dir)
	f := MustTempFile(dir)
	defer f.Close()

	w, err := tsm1.NewTSMWriter(f)
	if err != nil {
		t.Fatalf("unexpected error creating writer: %v", err)
	}

	var data = map[string][]tsm1.Value{
		"float": []tsm1.Value{
			tsm1.NewValue(1, 1.0)},
		"int": []tsm1.Value{
			tsm1.NewValue(1, int64(1))},
		"uint": []tsm1.Value{
			tsm1.NewValue(1, ^uint64(0))},
		"bool": []tsm1.Value{
			tsm1.NewValue(1, true)},
		"string": []tsm1.Value{
			tsm1.NewValue(1, "foo")},
	}

	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		if err := w.Write([]byte(k), data[k]); err != nil {
			t.Fatalf("unexpected error writing: %v", err)
		}
	}

	if err := w.WriteIndex(); err != nil {
		t.Fatalf("unexpected error writing index: %v", err)
	}

	if err := w.Close(); err != nil {
		t.Fatalf("unexpected error closing: %v", err)
	}

	f, err = os.Open(f.Name())
	if err != nil {
		t.Fatalf("unexpected error open file: %v", err)
	}

	r, err := tsm1.NewTSMReader(f)
	if err != nil {
		t.Fatalf("unexpected error created reader: %v", err)
	}
	defer r.Close()

	var count int
	for k, vals := range data {
		readValues, err := r.Read([]byte(k), vals[0].UnixNano())
		if err != nil {
			t.Fatalf("unexpected error readin: %v", err)
		}

		if exp, got := len(vals), len(readValues); exp != got {
			t.Fatalf("read values length mismatch: exp %v, got %v", exp, len(readValues))
		}

		for i, v := range vals {
			if v.Value() != readValues[i].Value() {
				t.Fatalf("read value mismatch(%d): exp %v, got %d", i, v.Value(), readValues[i].Value())
			}
		}
		count++
	}

	if exp, got := count, len(data); exp != got {
		t.Fatalf("read values count mismatch: exp %v, got %v", exp, got)
	}
}

func TestTSMReader_References(t *testing.T) {
	dir := MustTempDir()
	defer os.RemoveAll(dir)
	f := MustTempFile(dir)
	defer f.Close()

	w, err := tsm1.NewTSMWriter(f)
	if err != nil {
		t.Fatalf("unexpected error creating writer: %v", err)
	}

	var data = map[string][]tsm1.Value{
		"float": []tsm1.Value{
			tsm1.NewValue(1, 1.0)},
		"int": []tsm1.Value{
			tsm1.NewValue(1, int64(1))},
		"uint": []tsm1.Value{
			tsm1.NewValue(1, ^uint64(0))},
		"bool": []tsm1.Value{
			tsm1.NewValue(1, true)},
		"string": []tsm1.Value{
			tsm1.NewValue(1, "foo")},
	}

	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		if err := w.Write([]byte(k), data[k]); err != nil {
			t.Fatalf("unexpected error writing: %v", err)
		}
	}

	if err := w.WriteIndex(); err != nil {
		t.Fatalf("unexpected error writing index: %v", err)
	}

	if err := w.Close(); err != nil {
		t.Fatalf("unexpected error closing: %v", err)
	}

	f, err = os.Open(f.Name())
	if err != nil {
		t.Fatalf("unexpected error open file: %v", err)
	}

	r, err := tsm1.NewTSMReader(f)
	if err != nil {
		t.Fatalf("unexpected error created reader: %v", err)
	}
	defer r.Close()

	r.Ref()

	if err := r.Close(); err != tsm1.ErrFileInUse {
		t.Fatalf("expected error closing reader: %v", err)
	}

	if err := r.Remove(); err != tsm1.ErrFileInUse {
		t.Fatalf("expected error removing reader: %v", err)
	}

	var count int
	for k, vals := range data {
		readValues, err := r.Read([]byte(k), vals[0].UnixNano())
		if err != nil {
			t.Fatalf("unexpected error readin: %v", err)
		}

		if exp, got := len(vals), len(readValues); exp != got {
			t.Fatalf("read values length mismatch: exp %v, got %v", exp, len(readValues))
		}

		for i, v := range vals {
			if v.Value() != readValues[i].Value() {
				t.Fatalf("read value mismatch(%d): exp %v, got %d", i, v.Value(), readValues[i].Value())
			}
		}
		count++
	}

	if exp, got := count, len(data); exp != got {
		t.Fatalf("read values count mismatch: exp %v, got %v", exp, got)
	}
	r.Unref()

	if err := r.Close(); err != nil {
		t.Fatalf("unexpected error closing reader: %v", err)
	}

	if err := r.Remove(); err != nil {
		t.Fatalf("unexpected error removing reader: %v", err)
	}
}

func BenchmarkIndirectIndex_UnmarshalBinary(b *testing.B) {
	index := tsm1.NewIndexWriter()
	for i := 0; i < 100000; i++ {
		index.Add([]byte(fmt.Sprintf("cpu-%d", i)), tsm1.BlockFloat64, int64(i*2), int64(i*2+1), 10, 100)
	}

	bytes, err := index.MarshalBinary()
	if err != nil {
		b.Fatalf("unexpected error marshaling index: %v", err)
	}

	indirect := tsm1.NewIndirectIndex()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := indirect.UnmarshalBinary(bytes); err != nil {
			b.Fatalf("unexpected error unmarshaling index: %v", err)
		}
	}
}

func BenchmarkIndirectIndex_Entries(b *testing.B) {
	index := tsm1.NewIndexWriter()
	// add 1000 keys and 1000 blocks per key
	for i := 0; i < 1000; i++ {
		for j := 0; j < 1000; j++ {
			index.Add([]byte(fmt.Sprintf("cpu-%d", i)), tsm1.BlockFloat64, int64(i*j*2), int64(i*j*2+1), 10, 100)
		}
	}

	bytes, err := index.MarshalBinary()
	if err != nil {
		b.Fatalf("unexpected error marshaling index: %v", err)
	}

	indirect := tsm1.NewIndirectIndex()
	if err = indirect.UnmarshalBinary(bytes); err != nil {
		b.Fatalf("unexpected error unmarshaling index: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		indirect.Entries([]byte("cpu-1"))
	}
}

func BenchmarkIndirectIndex_ReadEntries(b *testing.B) {
	index := tsm1.NewIndexWriter()
	// add 1000 keys and 1000 blocks per key
	for i := 0; i < 1000; i++ {
		for j := 0; j < 1000; j++ {
			index.Add([]byte(fmt.Sprintf("cpu-%d", i)), tsm1.BlockFloat64, int64(i*j*2), int64(i*j*2+1), 10, 100)
		}
	}

	bytes, err := index.MarshalBinary()
	if err != nil {
		b.Fatalf("unexpected error marshaling index: %v", err)
	}

	var cache, entries []tsm1.IndexEntry
	indirect := tsm1.NewIndirectIndex()
	if err = indirect.UnmarshalBinary(bytes); err != nil {
		b.Fatalf("unexpected error unmarshaling index: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		entries = indirect.ReadEntries([]byte("cpu-1"), &cache)
	}

	b.Log(entries[0])
}
