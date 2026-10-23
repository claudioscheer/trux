package c

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestArenaBumpAllocatesSmallValuesFromOneChunk(t *testing.T) {
	compileAndRunRuntimeC(t, countingRuntimeSource()+`
int main(void) {
    rt_arena arena;
    rt_arena_init(&arena);

    void* first = rt_arena_alloc(&arena, 8);
    void* second = rt_arena_alloc(&arena, 8);
    if (first == NULL || second == NULL || first == second) {
        fprintf(stderr, "expected distinct non-null allocations\n");
        return 1;
    }
    if (rt_test_malloc_calls != 1) {
        fprintf(stderr, "malloc calls = %zu, want 1\n", rt_test_malloc_calls);
        return 1;
    }

    rt_arena_deinit(&arena);
    if (rt_test_free_calls != 1) {
        fprintf(stderr, "free calls = %zu, want 1\n", rt_test_free_calls);
        return 1;
    }
    return 0;
}
`)
}

func TestCheckedIntegerArithmeticSucceedsForValidEdges(t *testing.T) {
	compileAndRunRuntimeC(t, Source+`
int main(void) {
    if (rt_add_i64(INT64_MAX - 1, 1) != INT64_MAX) return 1;
    if (rt_sub_i64(INT64_MIN + 1, 1) != INT64_MIN) return 1;
    if (rt_mul_i64(INT64_MIN, 1) != INT64_MIN) return 1;
    if (rt_mul_i64(INT64_MAX, 1) != INT64_MAX) return 1;
    if (rt_div_i64(INT64_MIN, 1) != INT64_MIN) return 1;
    if (rt_rem_i64(17, 5) != 2) return 1;
    if (rt_rem_i64(-17, 5) != -2) return 1;
    return 0;
}
`)
}

func TestCheckedIntegerArithmeticFailsOnOverflow(t *testing.T) {
	tests := []struct {
		name string
		stmt string
		want string
	}{
		{name: "add", stmt: "(void)rt_add_i64(INT64_MAX, 1);", want: "integer overflow in +"},
		{name: "sub", stmt: "(void)rt_sub_i64(INT64_MIN, 1);", want: "integer overflow in -"},
		{name: "mul", stmt: "(void)rt_mul_i64(INT64_MAX, 2);", want: "integer overflow in *"},
		{name: "div_overflow", stmt: "(void)rt_div_i64(INT64_MIN, -1);", want: "integer overflow in /"},
		{name: "div_zero", stmt: "(void)rt_div_i64(1, 0);", want: "integer division by zero"},
		{name: "rem_zero", stmt: "(void)rt_rem_i64(1, 0);", want: "integer remainder by zero"},
		{name: "rem_overflow", stmt: "(void)rt_rem_i64(INT64_MIN, -1);", want: "integer overflow in %"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compileAndRunRuntimeCExpectFailure(t, Source+`
int main(void) {
    `+tt.stmt+`
    return 0;
}
`, tt.want)
		})
	}
}

func TestArenaResetReusesChunks(t *testing.T) {
	compileAndRunRuntimeC(t, countingRuntimeSource()+`
int main(void) {
    rt_arena arena;
    rt_arena_init(&arena);

    void* first = rt_arena_alloc(&arena, 8);
    rt_arena_reset(&arena);
    void* reused = rt_arena_alloc(&arena, 8);

    if (reused != first) {
        fprintf(stderr, "reset did not reuse chunk start\n");
        return 1;
    }
    if (rt_test_malloc_calls != 1) {
        fprintf(stderr, "malloc calls = %zu, want 1\n", rt_test_malloc_calls);
        return 1;
    }

    rt_arena_deinit(&arena);
    if (rt_test_free_calls != 1) {
        fprintf(stderr, "free calls = %zu, want 1\n", rt_test_free_calls);
        return 1;
    }
    return 0;
}
`)
}

func TestArenaRewindRestoresMarkedChunkOffset(t *testing.T) {
	compileAndRunRuntimeC(t, countingRuntimeSource()+`
int main(void) {
    rt_arena arena;
    rt_arena_init(&arena);

    void* first = rt_arena_alloc(&arena, 8);
    rt_arena_mark mark = rt_arena_mark_current(&arena);
    void* temporary = rt_arena_alloc(&arena, 8);
    rt_arena_rewind(&arena, mark);
    void* reused = rt_arena_alloc(&arena, 8);

    if (first == NULL || temporary == NULL || reused != temporary) {
        fprintf(stderr, "rewind did not restore marked offset\n");
        return 1;
    }
    if (rt_test_malloc_calls != 1) {
        fprintf(stderr, "malloc calls = %zu, want 1\n", rt_test_malloc_calls);
        return 1;
    }

    rt_arena_deinit(&arena);
    return 0;
}
`)
}

func TestArenaAllocatesLargeChunks(t *testing.T) {
	compileAndRunRuntimeC(t, countingRuntimeSource()+`
int main(void) {
    rt_arena arena;
    rt_arena_init(&arena);

    void* data = rt_arena_alloc(&arena, RT_ARENA_DEFAULT_CHUNK_SIZE + 1);
    if (data == NULL) {
        fprintf(stderr, "large allocation returned NULL\n");
        return 1;
    }
    if (arena.chunks == NULL || arena.chunks->cap < RT_ARENA_DEFAULT_CHUNK_SIZE + 1) {
        fprintf(stderr, "large allocation chunk too small\n");
        return 1;
    }

    rt_arena_deinit(&arena);
    return 0;
}
`)
}

func TestArenaRewindClearsListsRegisteredAfterMark(t *testing.T) {
	compileAndRunRuntimeC(t, countingRuntimeSource()+`
int main(void) {
    rt_arena arena;
    rt_arena_init(&arena);

    rt_list_int* keep = rt_list_int_from_values(&arena, (int64_t[]){1}, 1);
    rt_arena_mark mark = rt_arena_mark_current(&arena);
    rt_list_int* drop = rt_list_int_from_values(&arena, (int64_t[]){2}, 1);
    if (keep == NULL || drop == NULL || arena.lists == NULL || arena.lists->list != drop) {
        fprintf(stderr, "list setup failed\n");
        return 1;
    }

    rt_arena_rewind(&arena, mark);
    if (arena.lists == NULL || arena.lists->list != keep || arena.lists->next != NULL) {
        fprintf(stderr, "rewind did not restore list registration mark\n");
        return 1;
    }

    rt_arena_deinit(&arena);
    return 0;
}
`)
}

func TestArenaResetClearsRegisteredLists(t *testing.T) {
	compileAndRunRuntimeC(t, countingRuntimeSource()+`
int main(void) {
    rt_arena arena;
    rt_arena_init(&arena);

    rt_list_int* list = rt_list_int_from_values(&arena, (int64_t[]){1, 2}, 2);
    if (list == NULL || list->len != 2) {
        fprintf(stderr, "list setup failed\n");
        return 1;
    }
    rt_arena_reset(&arena);
    if (arena.lists != NULL) {
        fprintf(stderr, "reset did not clear registered lists\n");
        return 1;
    }

    rt_arena_deinit(&arena);
    return 0;
}
	`)
}

func TestCollectionCloneCopiesScalarData(t *testing.T) {
	compileAndRunRuntimeC(t, Source+`
int main(void) {
    rt_arena arena;
    rt_arena_init(&arena);

    rt_array_int source = rt_array_int_from_values(&arena, (int64_t[]){1, 2, 3}, 3);
    rt_array_int cloned = rt_array_int_clone(&arena, source);
    source.data[0] = 9;
    if (cloned.len != 3 || cloned.data[0] != 1) {
        fprintf(stderr, "array clone did not copy scalar data\n");
        return 1;
    }

    rt_slice_int slice = rt_array_int_slice(source, true, 1, false, 0);
    rt_slice_int cloned_slice = rt_slice_int_clone(&arena, slice);
    slice.data[0] = 8;
    if (cloned_slice.len != 2 || cloned_slice.data[0] != 2) {
        fprintf(stderr, "slice clone did not copy scalar data\n");
        return 1;
    }

    rt_arena_deinit(&arena);
    return 0;
}
`)
}

func TestStringCollectionCloneDeepCopiesElements(t *testing.T) {
	compileAndRunRuntimeC(t, Source+`
int main(void) {
    rt_arena source_arena;
    rt_arena target_arena;
    rt_arena_init(&source_arena);
    rt_arena_init(&target_arena);

    rt_string source_string = rt_string_clone(&source_arena, (rt_string){(const uint8_t*)"ab", 2});
    rt_array_string source = rt_array_string_from_values(&source_arena, (rt_string[]){source_string}, 1);
    rt_array_string cloned = rt_array_string_clone(&target_arena, source);
    if (cloned.data[0].data == source.data[0].data) {
        fprintf(stderr, "string collection clone reused element storage\n");
        return 1;
    }

    rt_arena_reset(&source_arena);
    if (!rt_string_equal(cloned.data[0], (rt_string){(const uint8_t*)"ab", 2})) {
        fprintf(stderr, "string collection clone did not preserve contents\n");
        return 1;
    }

    rt_arena_deinit(&target_arena);
    rt_arena_deinit(&source_arena);
    return 0;
}
`)
}

func TestNestedCollectionCloneDeepCopiesElements(t *testing.T) {
	compileAndRunRuntimeC(t, Source+`
#define RT_CLONE_VALUE_list_int(ARENA, VALUE) rt_list_int_clone((ARENA), (VALUE))
RT_DEFINE_COLLECTIONS(list_int, rt_list_int*)

int main(void) {
    rt_arena source_arena;
    rt_arena target_arena;
    rt_arena_init(&source_arena);
    rt_arena_init(&target_arena);

    rt_list_int* row = rt_list_int_from_values(&source_arena, (int64_t[]){1, 2}, 2);
    rt_list_list_int* rows = rt_list_list_int_from_values(&source_arena, (rt_list_int*[]){row}, 1);
    rt_list_list_int* cloned = rt_list_list_int_clone(&target_arena, rows);

    rt_list_int_set(row, 0, 9);
    if (rt_list_int_get(rt_list_list_int_get(cloned, 0), 0) != 1) {
        fprintf(stderr, "nested list clone reused inner storage\n");
        return 1;
    }

    rt_arena_reset(&source_arena);
    if (rt_list_int_get(rt_list_list_int_get(cloned, 0), 1) != 2) {
        fprintf(stderr, "nested list clone did not survive source arena reset\n");
        return 1;
    }

    rt_arena_deinit(&target_arena);
    rt_arena_deinit(&source_arena);
    return 0;
}
`)
}

func TestStringConcatCopiesEmptySideOperandsIntoTargetArena(t *testing.T) {
	compileAndRunRuntimeC(t, Source+`
int main(void) {
    rt_arena source_arena;
    rt_arena target_arena;
    rt_arena_init(&source_arena);
    rt_arena_init(&target_arena);

    rt_string source_string = rt_string_clone(&source_arena, (rt_string){(const uint8_t*)"ab", 2});
    rt_string right_empty = rt_string_concat(&target_arena, source_string, (rt_string){NULL, 0});
    rt_string left_empty = rt_string_concat(&target_arena, (rt_string){NULL, 0}, source_string);

    if (right_empty.data == source_string.data || left_empty.data == source_string.data) {
        fprintf(stderr, "concat reused source storage for empty-side operand\n");
        return 1;
    }

    rt_arena_reset(&source_arena);
    if (!rt_string_equal(right_empty, (rt_string){(const uint8_t*)"ab", 2}) ||
        !rt_string_equal(left_empty, (rt_string){(const uint8_t*)"ab", 2})) {
        fprintf(stderr, "concat result did not survive source arena reset\n");
        return 1;
    }

    rt_arena_deinit(&target_arena);
    rt_arena_deinit(&source_arena);
    return 0;
}
`)
}

func TestStringHelpersRejectInvalidNonEmptyString(t *testing.T) {
	tests := []struct {
		name string
		stmt string
	}{
		{
			name: "print",
			stmt: `rt_print_string((rt_string){NULL, 1});`,
		},
		{
			name: "clone",
			stmt: `rt_string ignored = rt_string_clone(&arena, (rt_string){NULL, 1}); (void)ignored;`,
		},
		{
			name: "equal",
			stmt: `bool ignored = rt_string_equal((rt_string){NULL, 1}, (rt_string){(const uint8_t*)"a", 1}); (void)ignored;`,
		},
		{
			name: "contains",
			stmt: `bool ignored = rt_string_contains((rt_string){(const uint8_t*)"a", 1}, (rt_string){NULL, 1}); (void)ignored;`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compileAndRunRuntimeCExpectFailure(t, Source+`
int main(void) {
    rt_arena arena;
    rt_arena_init(&arena);
    `+tt.stmt+`
    rt_arena_deinit(&arena);
    return 0;
}
`, "trux runtime error: invalid string: non-zero length with NULL data")
		})
	}
}

func TestCollectionHelpersRejectInvalidNonEmptyContainers(t *testing.T) {
	tests := []struct {
		name string
		stmt string
		want string
	}{
		{
			name: "array from values",
			stmt: `(void)rt_array_int_from_values(&arena, NULL, 1);`,
			want: "trux runtime error: invalid array values: non-zero length with NULL data",
		},
		{
			name: "array clone",
			stmt: `rt_array_int ignored = rt_array_int_clone(&arena, (rt_array_int){NULL, 1}); (void)ignored;`,
			want: "trux runtime error: invalid array: non-zero length with NULL data",
		},
		{
			name: "array get",
			stmt: `int64_t ignored = rt_array_int_get((rt_array_int){NULL, 1}, 0); (void)ignored;`,
			want: "trux runtime error: invalid array: non-zero length with NULL data",
		},
		{
			name: "array slice",
			stmt: `rt_slice_int ignored = rt_array_int_slice((rt_array_int){NULL, 1}, false, 0, false, 0); (void)ignored;`,
			want: "trux runtime error: invalid array: non-zero length with NULL data",
		},
		{
			name: "slice clone",
			stmt: `rt_slice_int ignored = rt_slice_int_clone(&arena, (rt_slice_int){NULL, 1}); (void)ignored;`,
			want: "trux runtime error: invalid slice: non-zero length with NULL data",
		},
		{
			name: "slice get",
			stmt: `int64_t ignored = rt_slice_int_get((rt_slice_int){NULL, 1}, 0); (void)ignored;`,
			want: "trux runtime error: invalid slice: non-zero length with NULL data",
		},
		{
			name: "slice slice",
			stmt: `rt_slice_int ignored = rt_slice_int_slice((rt_slice_int){NULL, 1}, false, 0, false, 0); (void)ignored;`,
			want: "trux runtime error: invalid slice: non-zero length with NULL data",
		},
		{
			name: "list from values",
			stmt: `(void)rt_list_int_from_values(&arena, NULL, 1);`,
			want: "trux runtime error: invalid list values: non-zero length with NULL data",
		},
		{
			name: "list clone",
			stmt: `rt_list_int bad = {NULL, 1, 1}; rt_list_int* ignored = rt_list_int_clone(&arena, &bad); (void)ignored;`,
			want: "trux runtime error: invalid list: non-zero length with NULL data",
		},
		{
			name: "list get",
			stmt: `rt_list_int bad = {NULL, 1, 1}; int64_t ignored = rt_list_int_get(&bad, 0); (void)ignored;`,
			want: "trux runtime error: invalid list: non-zero length with NULL data",
		},
		{
			name: "list slice",
			stmt: `rt_list_int bad = {NULL, 1, 1}; rt_slice_int ignored = rt_list_int_slice(&bad, false, 0, false, 0); (void)ignored;`,
			want: "trux runtime error: invalid list: non-zero length with NULL data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compileAndRunRuntimeCExpectFailure(t, Source+`
int main(void) {
    rt_arena arena;
    rt_arena_init(&arena);
    `+tt.stmt+`
    rt_arena_deinit(&arena);
    return 0;
}
`, tt.want)
		})
	}
}

func TestCheckedLenI64RejectsTooLargeLength(t *testing.T) {
	compileAndRunRuntimeCExpectFailure(t, Source+`
int main(void) {
    (void)rt_checked_len_i64((size_t)INT64_MAX + (size_t)1);
    return 0;
}
`, "trux runtime error: length too large")
}

func TestListCloneRegistersWithTargetArena(t *testing.T) {
	compileAndRunRuntimeC(t, Source+`
int main(void) {
    rt_arena source_arena;
    rt_arena target_arena;
    rt_arena_init(&source_arena);
    rt_arena_init(&target_arena);

    rt_list_int* source = rt_list_int_from_values(&source_arena, (int64_t[]){1, 2}, 2);
    rt_list_int* cloned = rt_list_int_clone(&target_arena, source);
    if (target_arena.lists == NULL || target_arena.lists->list != cloned) {
        fprintf(stderr, "list clone was not registered with target arena\n");
        return 1;
    }

    rt_arena_deinit(&target_arena);
    rt_arena_deinit(&source_arena);
    return 0;
}
`)
}

func TestInputRuntimeHelpersReadLinesAndTypedValues(t *testing.T) {
	compileAndRunRuntimeCWithInput(t, Source+`
int main(void) {
    rt_arena arena;
    rt_arena_init(&arena);

    rt_string line = rt_read_line(&arena);
    int64_t count = rt_read_int(&arena);
    double ratio = rt_read_float(&arena);
    bool ready = rt_read_bool(&arena);

    if (!rt_string_equal(line, (rt_string){(const uint8_t*)"hello", 5}) ||
        count != 42 ||
        ratio != 2.5 ||
        !ready) {
        fprintf(stderr, "input helpers returned unexpected values\n");
        return 1;
    }

    rt_arena_deinit(&arena);
    return 0;
}
`, "hello\r\n42\n2.5\ntrue\n")
}

func TestFileAndCSVRuntimeHelpers(t *testing.T) {
	dir := t.TempDir()
	textPath := filepath.Join(dir, "text.txt")
	csvPath := filepath.Join(dir, "in.csv")
	outCSVPath := filepath.Join(dir, "out.csv")
	source := Source + `
int main(void) {
    rt_arena arena;
    rt_arena_init(&arena);

    rt_string text_path = ` + cString(textPath) + `;
    rt_string csv_path = ` + cString(csvPath) + `;
    rt_string out_csv_path = ` + cString(outCSVPath) + `;

    rt_error write_err = rt_write_file(&arena, text_path, (rt_string){(const uint8_t*)"alpha\nbeta", 10});
    if (write_err != NULL) {
        fprintf(stderr, "rt_write_file failed\n");
        return 1;
    }
    trux_tuple_string_error text_res = rt_read_file(&arena, text_path);
    if (text_res.f1 != NULL) {
        fprintf(stderr, "rt_read_file returned unexpected error\n");
        return 1;
    }
    rt_string text = text_res.f0;
    if (!rt_string_equal(text, (rt_string){(const uint8_t*)"alpha\nbeta", 10})) {
        fprintf(stderr, "rt_read_file returned unexpected contents\n");
        return 1;
    }

    write_err = rt_write_file(&arena, csv_path, (rt_string){(const uint8_t*)"name,score\n\"A, B\",2\n", 20});
    if (write_err != NULL) {
        fprintf(stderr, "rt_write_file csv failed\n");
        return 1;
    }
    trux_tuple_list_string_error csv_res = rt_read_csv(&arena, csv_path, 2);
    if (csv_res.f1 != NULL) {
        fprintf(stderr, "rt_read_csv returned unexpected error\n");
        return 1;
    }
    rt_list_string* cells = csv_res.f0;
    if (cells->len != 4 ||
        !rt_string_equal(cells->data[0], (rt_string){(const uint8_t*)"name", 4}) ||
        !rt_string_equal(cells->data[2], (rt_string){(const uint8_t*)"A, B", 4})) {
        fprintf(stderr, "rt_read_csv returned unexpected cells\n");
        return 1;
    }

    rt_error csv_err = rt_write_csv(&arena, out_csv_path, cells, 2);
    if (csv_err != NULL) {
        fprintf(stderr, "rt_write_csv returned unexpected error\n");
        return 1;
    }
    rt_arena_deinit(&arena);
    return 0;
}
`

	compileAndRunRuntimeC(t, source)
	got, err := os.ReadFile(outCSVPath)
	if err != nil {
		t.Fatal(err)
	}
	want := "name,score\n\"A, B\",2\n"
	if string(got) != want {
		t.Fatalf("written CSV = %q, want %q", got, want)
	}
}

func TestRuntimeFilesystemOperations(t *testing.T) {
	tempDir := t.TempDir()
	subDir := filepath.Join(tempDir, "test_dir")
	fileA := filepath.Join(subDir, "file_a.txt")
	fileB := filepath.Join(subDir, "file_b.txt")

	source := Source + `
int main(void) {
    rt_arena arena;
    rt_arena_init(&arena);

    rt_string dir_path = ` + cString(subDir) + `;
    rt_string file_a_path = ` + cString(fileA) + `;
    rt_string file_b_path = ` + cString(fileB) + `;

    if (rt_exists(dir_path)) {
        fprintf(stderr, "dir should not exist initially\n");
        return 1;
    }
    if (rt_is_dir(dir_path)) {
        fprintf(stderr, "dir should not be dir initially\n");
        return 1;
    }
    if (rt_is_file(dir_path)) {
        fprintf(stderr, "dir should not be file initially\n");
        return 1;
    }

    rt_error err = rt_create_dir(&arena, dir_path);
    if (err != NULL) {
        fprintf(stderr, "rt_create_dir failed\n");
        return 1;
    }
    if (!rt_exists(dir_path)) {
        fprintf(stderr, "dir should exist after createDir\n");
        return 1;
    }
    if (!rt_is_dir(dir_path)) {
        fprintf(stderr, "dir should be dir\n");
        return 1;
    }
    if (rt_is_file(dir_path)) {
        fprintf(stderr, "dir should not be regular file\n");
        return 1;
    }

    err = rt_create_dir(&arena, dir_path);
    if (err == NULL) {
        fprintf(stderr, "createDir on existing path should return error\n");
        return 1;
    }

    err = rt_write_file(&arena, file_b_path, (rt_string){(const uint8_t*)"b", 1});
    if (err != NULL) {
        fprintf(stderr, "write file_b failed\n");
        return 1;
    }
    err = rt_write_file(&arena, file_a_path, (rt_string){(const uint8_t*)"a", 1});
    if (err != NULL) {
        fprintf(stderr, "write file_a failed\n");
        return 1;
    }

    if (!rt_is_file(file_a_path)) {
        fprintf(stderr, "file_a should be file\n");
        return 1;
    }
    if (rt_is_dir(file_a_path)) {
        fprintf(stderr, "file_a should not be dir\n");
        return 1;
    }

    trux_tuple_list_string_error dir_res = rt_read_dir(&arena, dir_path);
    if (dir_res.f1 != NULL) {
        fprintf(stderr, "rt_read_dir failed\n");
        return 1;
    }
    rt_list_string* entries = dir_res.f0;
    if (entries->len != 2) {
        fprintf(stderr, "expected 2 entries, got %zu\n", entries->len);
        return 1;
    }
    if (!rt_string_equal(entries->data[0], (rt_string){(const uint8_t*)"file_a.txt", 10})) {
        fprintf(stderr, "first entry should be file_a.txt\n");
        return 1;
    }
    if (!rt_string_equal(entries->data[1], (rt_string){(const uint8_t*)"file_b.txt", 10})) {
        fprintf(stderr, "second entry should be file_b.txt\n");
        return 1;
    }

    err = rt_remove(&arena, dir_path);
    if (err == NULL) {
        fprintf(stderr, "rt_remove on non-empty dir should fail\n");
        return 1;
    }

    err = rt_remove(&arena, file_a_path);
    if (err != NULL) {
        fprintf(stderr, "remove file_a failed\n");
        return 1;
    }
    err = rt_remove(&arena, file_b_path);
    if (err != NULL) {
        fprintf(stderr, "remove file_b failed\n");
        return 1;
    }

    err = rt_remove(&arena, dir_path);
    if (err != NULL) {
        fprintf(stderr, "remove empty dir failed\n");
        return 1;
    }
    if (rt_exists(dir_path)) {
        fprintf(stderr, "dir should not exist after remove\n");
        return 1;
    }

    rt_arena_deinit(&arena);
    return 0;
}
`
	compileAndRunRuntimeC(t, source)
}

func TestCSVRuntimeRejectsWrongColumnCount(t *testing.T) {
	dir := t.TempDir()
	csvPath := filepath.Join(dir, "bad.csv")
	if err := os.WriteFile(csvPath, []byte("a,b\nc\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	compileAndRunRuntimeC(t, Source+`
int main(void) {
    rt_arena arena;
    rt_arena_init(&arena);
    trux_tuple_list_string_error res = rt_read_csv(&arena, `+cString(csvPath)+`, 2);
    if (res.f1 == NULL) {
        fprintf(stderr, "expected error on bad column count\n");
        return 1;
    }
    if (!rt_string_equal(res.f1->message, (rt_string){(const uint8_t*)"csv row column count mismatch", 29})) {
        fprintf(stderr, "unexpected error message\n");
        return 2;
    }
    rt_arena_deinit(&arena);
    return 0;
}
`)
}

func TestFrameChunkPoolReusesChunksAndStaysContextLocal(t *testing.T) {
	compileAndRunRuntimeC(t, countingRuntimeSource()+`
static int recurse(rt_context* ctx, int n) {
    rt_arena frame;
    rt_arena_init_frame(&frame, ctx);
    unsigned char* bytes = (unsigned char*)rt_arena_alloc(&frame, 16);
    if (bytes == NULL) {
        rt_arena_deinit(&frame);
        return 0;
    }
    bytes[0] = (unsigned char)n;
    int ok = 1;
    if (n > 0) {
        ok = recurse(ctx, n - 1);
    }
    if (bytes[0] != (unsigned char)n) {
        ok = 0;
    }
    rt_arena_deinit(&frame);
    return ok;
}

int main(void) {
    rt_context pooled = {0};
    size_t before = rt_test_malloc_calls;
    int i;
    for (i = 0; i < 40; i++) {
        rt_arena frame;
        rt_arena_init_frame(&frame, &pooled);
        if (rt_arena_alloc(&frame, 64) == NULL) return 1;
        rt_arena_deinit(&frame);
    }
    size_t pooled_mallocs = rt_test_malloc_calls - before;
    rt_context_free_chunks(&pooled);

    before = rt_test_malloc_calls;
    for (i = 0; i < 40; i++) {
        rt_arena frame;
        rt_arena_init(&frame);
        if (rt_arena_alloc(&frame, 64) == NULL) return 1;
        rt_arena_deinit(&frame);
    }
    size_t fresh_mallocs = rt_test_malloc_calls - before;
    if (!(pooled_mallocs < fresh_mallocs)) {
        fprintf(stderr, "pooled %zu fresh %zu\n", pooled_mallocs, fresh_mallocs);
        return 2;
    }

    rt_context first_ctx = {0};
    rt_context second_ctx = {0};
    rt_arena frame;
    rt_arena_init_frame(&frame, &first_ctx);
    void* first = rt_arena_alloc(&frame, 32);
    rt_arena_deinit(&frame);
    rt_arena_init_frame(&frame, &second_ctx);
    void* second = rt_arena_alloc(&frame, 32);
    if (second == first) return 3;
    rt_arena_deinit(&frame);
    rt_context_free_chunks(&first_ctx);
    rt_context_free_chunks(&second_ctx);

    rt_context recursive = {0};
    if (!recurse(&recursive, 8)) return 4;
    rt_context_free_chunks(&recursive);
    return 0;
}
`)
}

func TestListHeaderSharesRegistrationAllocation(t *testing.T) {
	compileAndRunRuntimeC(t, countingRuntimeSource()+`
int main(void) {
    rt_arena arena;
    rt_arena_init(&arena);

    size_t before = rt_test_malloc_calls;
    rt_list_int* list = rt_list_int_new(&arena, 4);
    size_t created = rt_test_malloc_calls - before;
    if (created != 2) {
        fprintf(stderr, "list mallocs = %zu, want 2\n", created);
        return 1;
    }
    uintptr_t node = (uintptr_t)arena.lists;
    uintptr_t header = (uintptr_t)list;
    if (arena.lists == NULL || arena.lists->list != list ||
        header < node + sizeof(rt_list_allocation) ||
        header >= node + sizeof(rt_list_allocation) + 64) {
        fprintf(stderr, "list header is not inside its registration block\n");
        return 2;
    }

    rt_list_int_append(list, 10);
    rt_list_int_append(list, 20);
    rt_list_int_set(list, 0, 7);
    if (rt_list_int_get(list, 0) != 7 || rt_list_int_get(list, 1) != 20 || list->len != 2) {
        fprintf(stderr, "list append or index failed\n");
        return 3;
    }

    rt_list_int* cloned = rt_list_int_clone(&arena, list);
    rt_list_int_set(list, 0, 1);
    if (cloned == list || rt_list_int_get(cloned, 0) != 7 || rt_list_int_get(cloned, 1) != 20) {
        fprintf(stderr, "list clone did not keep an independent copy\n");
        return 4;
    }

    before = rt_test_malloc_calls;
    rt_list_int* empty = rt_list_int_new(&arena, 0);
    if (rt_test_malloc_calls - before != 1) {
        fprintf(stderr, "empty list mallocs = %zu, want 1\n", rt_test_malloc_calls - before);
        return 5;
    }
    uintptr_t empty_node = (uintptr_t)arena.lists;
    uintptr_t empty_header = (uintptr_t)empty;
    if (arena.lists->list != empty ||
        empty_header < empty_node + sizeof(rt_list_allocation) ||
        empty_header >= empty_node + sizeof(rt_list_allocation) + 64) {
        fprintf(stderr, "empty list header is not inside its registration block\n");
        return 6;
    }
    rt_list_int_append(empty, 5);
    if (rt_list_int_get(empty, 0) != 5 || empty->len != 1) {
        fprintf(stderr, "append on an empty list failed\n");
        return 7;
    }

    rt_arena_deinit(&arena);
    if (arena.lists != NULL) {
        fprintf(stderr, "deinit left a list registration\n");
        return 8;
    }
    return 0;
}
`)
}

func TestReadFileFillsOneBuffer(t *testing.T) {
	dir := t.TempDir()
	smallPath := filepath.Join(dir, "small.bin")
	small := []byte{0, 1, 255, 'a', '\n', 0}
	if err := os.WriteFile(smallPath, small, 0o644); err != nil {
		t.Fatal(err)
	}
	emptyPath := filepath.Join(dir, "empty.bin")
	if err := os.WriteFile(emptyPath, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	bigPath := filepath.Join(dir, "big.bin")
	if err := os.WriteFile(bigPath, bytes.Repeat([]byte{'Z'}, 5000), 0o644); err != nil {
		t.Fatal(err)
	}

	compileAndRunRuntimeC(t, countingRuntimeSource()+`
int main(void) {
    rt_arena arena;
    rt_arena_init(&arena);

    rt_string small_path = `+cString(smallPath)+`;
    rt_string empty_path = `+cString(emptyPath)+`;
    rt_string big_path = `+cString(bigPath)+`;
    const uint8_t small_want[] = {0, 1, 255, 'a', 10, 0};

    size_t before = rt_test_malloc_calls;
    trux_tuple_string_error text_res = rt_read_file(&arena, small_path);
    if (text_res.f1 != NULL) {
        fprintf(stderr, "small read returned error\n");
        return 1;
    }
    rt_string text = text_res.f0;
    if (rt_test_malloc_calls - before != 2) {
        fprintf(stderr, "small read mallocs = %zu, want 2\n", rt_test_malloc_calls - before);
        return 1;
    }
    if (arena.chunks == NULL || arena.chunks->next != NULL) {
        fprintf(stderr, "small read did not use exactly one chunk\n");
        return 2;
    }
    if (text.len != 6 || text.data == NULL || memcmp(text.data, small_want, 6) != 0) {
        fprintf(stderr, "small read bytes mismatch\n");
        return 3;
    }
    uintptr_t chunk_start = (uintptr_t)arena.chunks->data;
    uintptr_t chunk_end = chunk_start + arena.chunks->used;
    uintptr_t text_addr = (uintptr_t)text.data;
    if (text_addr < chunk_start || text_addr + text.len > chunk_end) {
        fprintf(stderr, "small read is not inside the arena chunk\n");
        return 4;
    }

    before = rt_test_malloc_calls;
    trux_tuple_string_error empty_res = rt_read_file(&arena, empty_path);
    if (empty_res.f1 != NULL) {
        fprintf(stderr, "empty read returned error\n");
        return 5;
    }
    rt_string empty = empty_res.f0;
    if (rt_test_malloc_calls - before != 1) {
        fprintf(stderr, "empty read mallocs = %zu, want 1\n", rt_test_malloc_calls - before);
        return 5;
    }
    if (empty.len != 0 || empty.data != NULL) {
        fprintf(stderr, "empty read was not an empty string\n");
        return 6;
    }
    if (arena.chunks == NULL || arena.chunks->next != NULL) {
        fprintf(stderr, "empty read allocated another chunk\n");
        return 7;
    }

    before = rt_test_malloc_calls;
    trux_tuple_string_error big_res = rt_read_file(&arena, big_path);
    if (big_res.f1 != NULL) {
        fprintf(stderr, "big read returned error\n");
        return 8;
    }
    rt_string big = big_res.f0;
    if (rt_test_malloc_calls - before != 2) {
        fprintf(stderr, "big read mallocs = %zu, want 2\n", rt_test_malloc_calls - before);
        return 8;
    }
    if (big.len != 5000 || big.data == NULL || big.data[0] != 'Z' || big.data[4999] != 'Z') {
        fprintf(stderr, "big read bytes mismatch\n");
        return 9;
    }
    if (arena.chunks == NULL || arena.chunks->cap < 5000) {
        fprintf(stderr, "big read chunk cap = %zu\n", arena.chunks == NULL ? 0 : arena.chunks->cap);
        return 10;
    }

    rt_arena_deinit(&arena);
    return 0;
}
`)
}

func TestArenaChunksGrowGeometrically(t *testing.T) {
	compileAndRunRuntimeC(t, countingRuntimeSource()+`
int main(void) {
    rt_arena arena;
    rt_arena_init(&arena);

    if (rt_arena_alloc(&arena, RT_ARENA_DEFAULT_CHUNK_SIZE) == NULL) {
        return 1;
    }
    size_t first_cap = arena.chunks->cap;
    if (first_cap != RT_ARENA_DEFAULT_CHUNK_SIZE) {
        fprintf(stderr, "first cap = %zu\n", first_cap);
        return 2;
    }

    if (rt_arena_alloc(&arena, 1) == NULL) {
        return 1;
    }
    size_t second_cap = arena.chunks->cap;
    if (second_cap < first_cap * 2) {
        fprintf(stderr, "second cap = %zu, first = %zu\n", second_cap, first_cap);
        return 3;
    }

    size_t room = arena.chunks->cap - arena.chunks->used;
    if (rt_arena_alloc(&arena, room) == NULL) {
        return 1;
    }
    if (arena.chunks->cap != second_cap || arena.chunks->used != second_cap) {
        fprintf(stderr, "second chunk was not filled in place\n");
        return 4;
    }

    if (rt_arena_alloc(&arena, 1) == NULL) {
        return 1;
    }
    size_t third_cap = arena.chunks->cap;
    if (third_cap < second_cap * 2) {
        fprintf(stderr, "third cap = %zu, second = %zu\n", third_cap, second_cap);
        return 5;
    }
    if (rt_test_malloc_calls != 3) {
        fprintf(stderr, "chunk mallocs = %zu, want 3\n", rt_test_malloc_calls);
        return 6;
    }

    size_t used = first_cap + second_cap + (size_t)RT_ALIGNOF(max_align_t);
    size_t fixed_chunks = (used + RT_ARENA_DEFAULT_CHUNK_SIZE - 1) / RT_ARENA_DEFAULT_CHUNK_SIZE;
    if (!(rt_test_malloc_calls < fixed_chunks)) {
        fprintf(stderr, "geometric mallocs = %zu, fixed 4KB chunks = %zu\n", rt_test_malloc_calls, fixed_chunks);
        return 7;
    }

    rt_arena_deinit(&arena);

    rt_context ctx = {0};
    rt_arena frame;
    rt_arena_init_frame(&frame, &ctx);
    if (rt_arena_alloc(&frame, RT_ARENA_DEFAULT_CHUNK_SIZE) == NULL || rt_arena_alloc(&frame, 1) == NULL) {
        return 1;
    }
    rt_arena_deinit(&frame);
    size_t pooled = rt_test_malloc_calls;
    rt_arena_init_frame(&frame, &ctx);
    if (rt_arena_alloc(&frame, RT_ARENA_DEFAULT_CHUNK_SIZE) == NULL || rt_arena_alloc(&frame, 1) == NULL) {
        return 1;
    }
    if (rt_test_malloc_calls != pooled) {
        fprintf(stderr, "pool reused %zu then allocated %zu\n", pooled, rt_test_malloc_calls);
        return 8;
    }
    rt_arena_deinit(&frame);
    rt_context_free_chunks(&ctx);
    return 0;
}
`)
}

func countingRuntimeSource() string {
	const stdlibInclude = "#include <stdlib.h>\n"
	const mallocHooks = `#include <stdlib.h>

static size_t rt_test_malloc_calls = 0;
static size_t rt_test_free_calls = 0;

static void* rt_test_malloc(size_t size) {
    rt_test_malloc_calls++;
    return malloc(size);
}

static void rt_test_free(void* ptr) {
    rt_test_free_calls++;
    free(ptr);
}

static void* rt_test_realloc(void* ptr, size_t size) {
    return realloc(ptr, size);
}

#define malloc rt_test_malloc
#define free rt_test_free
#define realloc rt_test_realloc
`
	return strings.Replace(Source, stdlibInclude, mallocHooks, 1)
}

func compileAndRunRuntimeC(t *testing.T, source string) {
	t.Helper()

	exePath := compileRuntimeC(t, source)
	run := exec.Command(exePath)
	output, err := run.CombinedOutput()
	if err != nil {
		t.Fatalf("run runtime C: %v\n%s", err, output)
	}
}

func compileAndRunRuntimeCWithInput(t *testing.T, source string, input string) {
	t.Helper()

	exePath := compileRuntimeC(t, source)
	run := exec.Command(exePath)
	run.Stdin = strings.NewReader(input)
	output, err := run.CombinedOutput()
	if err != nil {
		t.Fatalf("run runtime C: %v\n%s", err, output)
	}
}

func compileAndRunRuntimeCExpectFailure(t *testing.T, source string, want string) {
	t.Helper()

	exePath := compileRuntimeC(t, source)
	run := exec.Command(exePath)
	output, err := run.CombinedOutput()
	if err == nil {
		t.Fatalf("run runtime C succeeded, want failure containing %q\n%s", want, output)
	}
	if !strings.Contains(string(output), want) {
		t.Fatalf("run runtime C output = %q, want it to contain %q", output, want)
	}
}

func compileRuntimeC(t *testing.T, source string) string {
	t.Helper()

	compiler := os.Getenv("CC")
	if compiler == "" {
		compiler = "cc"
	}
	if _, err := exec.LookPath(compiler); err != nil {
		t.Skipf("C compiler %q not found", compiler)
	}

	dir := t.TempDir()
	cPath := filepath.Join(dir, "main.c")
	exePath := filepath.Join(dir, "main")
	if err := os.WriteFile(cPath, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}

	compile := exec.Command(compiler, "-std=c11", "-Wall", "-Wextra", "-pedantic", cPath, "-o", exePath)
	output, err := compile.CombinedOutput()
	if err != nil {
		t.Fatalf("compile runtime C: %v\n%s", err, output)
	}
	if strings.TrimSpace(string(output)) != "" {
		t.Fatalf("strict C compiler emitted warnings:\n%s", output)
	}

	return exePath
}

func cString(value string) string {
	return "(rt_string){(const uint8_t*)" + strconv.Quote(value) + ", " + strconv.Itoa(len(value)) + "}"
}

func TestNilPointerDereferenceCheckFailsCleanly(t *testing.T) {
	compileAndRunRuntimeCExpectFailure(t, Source+`
int main(void) {
    int64_t* p = NULL;
    int64_t v = *((int64_t*)rt_check_not_null_at(p, "main.tx", 10, 5));
    (void)v;
    return 0;
}
`, "trux runtime error: nil pointer dereference")
}

func TestMapOperationsPreserveOrderAndFreeInArena(t *testing.T) {
	compileAndRunRuntimeC(t, countingRuntimeSource()+`
RT_DEFINE_MAP(string_int, string, rt_string, int, int64_t, rt_eq_string, rt_hash_string, 0)
RT_DEFINE_MAP_LISTS(string_int, string, rt_string, int, int64_t)

int main(void) {
    rt_arena arena;
    rt_arena_init(&arena);

    rt_map_string_int* m = rt_map_string_int_new(&arena, 2);
    rt_string k1 = `+cString("first")+`;
    rt_string k2 = `+cString("second")+`;
    rt_string k3 = `+cString("third")+`;

    rt_map_string_int_set(m, k1, 10);
    rt_map_string_int_set(m, k2, 20);
    rt_map_string_int_set(m, k3, 30);
    rt_map_string_int_set(m, k2, 25);

    if (rt_map_string_int_len(m) != 3) return 1;
    if (rt_map_string_int_get(m, k2) != 25) return 2;
    if (rt_map_string_int_get(m, `+cString("missing")+`) != 0) return 3;

    bool ok = false;
    int64_t found = rt_map_string_int_get_ok(m, k3, &ok);
    if (!ok || found != 30) return 4;
    int64_t missing = rt_map_string_int_get_ok(m, `+cString("missing")+`, &ok);
    if (ok || missing != 0) return 5;

    rt_map_string_int_delete(m, k1);
    if (rt_map_string_int_len(m) != 2) return 6;

    rt_list_string* ks = rt_map_string_int_keys(&arena, m);
    rt_list_int* vs = rt_map_string_int_values(&arena, m);
    if (ks->len != 2 || vs->len != 2) return 7;
    if (!rt_string_equal(ks->data[0], k2) || !rt_string_equal(ks->data[1], k3)) return 8;
    if (vs->data[0] != 25 || vs->data[1] != 30) return 9;

    rt_map_string_int* copy = rt_map_string_int_clone(&arena, m);
    rt_map_string_int_set(copy, k2, 99);
    if (rt_map_string_int_get(m, k2) != 25 || rt_map_string_int_get(copy, k2) != 99) return 10;

    rt_arena_deinit(&arena);
    if (rt_test_free_calls != rt_test_malloc_calls) {
        fprintf(stderr, "malloc=%zu free=%zu\n", rt_test_malloc_calls, rt_test_free_calls);
        return 11;
    }
    return 0;
}
`)
}

type envPair struct {
	key   string
	value string
}

func TestEnvParseFileAndProcessEntries(t *testing.T) {
	exe := compileRuntimeC(t, Source+envParseDriver)

	t.Run("file bytes", func(t *testing.T) {
		input, want := envFileFixture()
		got := parseEnvFixture(t, exe, input, false)
		assertEnvPairs(t, got, want)
	})
	t.Run("empty", func(t *testing.T) {
		got := parseEnvFixture(t, exe, nil, false)
		assertEnvPairs(t, got, nil)
	})
	t.Run("comments and whitespace", func(t *testing.T) {
		got := parseEnvFixture(t, exe, []byte("# hidden=1\n\n  \n\t\n  # x\n"), false)
		assertEnvPairs(t, got, nil)
	})
	t.Run("carriage return is not a line break", func(t *testing.T) {
		got := parseEnvFixture(t, exe, []byte("OK=1\rBAD=2"), false)
		assertEnvPairs(t, got, []envPair{{"OK", "1\rBAD=2"}})
	})
	t.Run("nul in value", func(t *testing.T) {
		input := append([]byte("NUL="), 'a', 0, 'b')
		input = append(input, '\n', 'N', 'E', 'X', 'T', '=', 'z')
		got := parseEnvFixture(t, exe, input, false)
		assertEnvPairs(t, got, []envPair{{"NUL", string([]byte{'a', 0, 'b'})}, {"NEXT", "z"}})
	})
	t.Run("process entries", func(t *testing.T) {
		got := parseEnvFixture(t, exe, nil, true)
		assertEnvPairs(t, got, syntheticEnvPairs())
	})
	t.Run("file overlays process slot", func(t *testing.T) {
		input := []byte("# comment\n\nBOTH=from-file\nNEW=new\n")
		got := parseEnvFixture(t, exe, input, true)
		want := append(append([]envPair{}, syntheticEnvPairs()...), envPair{"NEW", "new"})
		want[1].value = "from-file"
		assertEnvPairs(t, got, want)
	})

	rejects := []struct {
		name string
		body string
		want string
	}{
		{name: "no equals", body: "not valid\n", want: ".env:1: invalid line"},
		{name: "export", body: "export FOO=bar\n", want: ".env:1: invalid key"},
		{name: "digit", body: "1FOO=bar\n", want: ".env:1: invalid key"},
		{name: "hyphen", body: "FOO-BAR=1\n", want: ".env:1: invalid key"},
		{name: "dot", body: "FOO.BAR=1\n", want: ".env:1: invalid key"},
		{name: "empty key", body: "=value\n", want: ".env:1: invalid key"},
		{name: "blank key", body: "   =x\n", want: ".env:1: invalid key"},
		{name: "space in key", body: "FOO BAR=1\n", want: ".env:1: invalid key"},
		{name: "tab in key", body: "FOO\tBAR=1\n", want: ".env:1: invalid key"},
		{name: "quoted key", body: "\"FOO\"=bar\n", want: ".env:1: invalid key"},
		{name: "unicode key", body: "Ä=1\n", want: ".env:1: invalid key"},
		{name: "later line", body: "# comment\n\nOK=1\nnot valid\n", want: ".env:4: invalid line"},
		{name: "crlf line", body: "OK=1\r\n\r\nnot valid\r\n", want: ".env:3: invalid line"},
		{name: "key on line 2", body: "OK=1\n1NO=x\n", want: ".env:2: invalid key"},
	}
	for _, tt := range rejects {
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr, err := runEnvFixture(t, exe, []byte(tt.body), false)
			if err == nil {
				t.Fatalf("parse succeeded, want %s\n%s", tt.want, stdout)
			}
			if len(stdout) != 0 {
				t.Fatalf("stdout = %q, want empty", stdout)
			}
			if !strings.Contains(err.Error(), "exit status 1") {
				t.Fatalf("error = %q, want exit status 1", err.Error())
			}
			if !bytes.Contains(stderr, []byte("trux runtime error: "+tt.want)) {
				t.Fatalf("stderr = %q, want %q", stderr, tt.want)
			}
		})
	}
}

func syntheticEnvPairs() []envPair {
	return []envPair{
		{"PATH", "from-proc"},
		{"BOTH", "from-proc"},
		{"ProgramFiles(x86)", `C:\PF`},
		{"EMPTYPROC", ""},
		{"EQ", "a=b"},
		{"A B", "kept"},
	}
}

func envFileFixture() ([]byte, []envPair) {
	lines := []string{
		"# comment",
		"",
		"  # indented",
		"   ",
		"\t",
		"\tTAB\t=tabbed",
		"CLEAR=",
		`QUOTED="hello"`,
		"DOLLAR=$PROC",
		"A1=a1",
		"_UNDER=under",
		"_=blank",
		"EQS===",
		"TRAIL=end" + "  ",
		"PAD = spaced",
		"HASH=#value",
		"UTF=caf\u00e9",
		"PRE=short",
		"PRE_FIX=long",
		"LONG=" + strings.Repeat("x", 300),
		"N0=old",
	}
	want := []envPair{
		{"TAB", "tabbed"},
		{"CLEAR", ""},
		{"QUOTED", `"hello"`},
		{"DOLLAR", "$PROC"},
		{"A1", "a1"},
		{"_UNDER", "under"},
		{"_", "blank"},
		{"EQS", "=="},
		{"TRAIL", "end  "},
		{"PAD", " spaced"},
		{"HASH", "#value"},
		{"UTF", "caf\u00e9"},
		{"PRE", "short"},
		{"PRE_FIX", "long"},
		{"LONG", strings.Repeat("x", 300)},
		{"N0", "v0"},
	}
	for i := 1; i <= 9; i++ {
		lines = append(lines, fmt.Sprintf("N%d=v%d", i, i))
	}
	lines = append(lines, "N0=v0", "SPACES="+strings.Repeat(" ", 3))
	for i := 1; i <= 9; i++ {
		want = append(want, envPair{fmt.Sprintf("N%d", i), fmt.Sprintf("v%d", i)})
	}
	want = append(want, envPair{"SPACES", "   "}, envPair{"CR", "cr"}, envPair{"END", "end"})
	body := strings.Join(lines, "\n") + "\n" + "CR=cr\r\n" + "END=end\r"
	return []byte(body), want
}

func parseEnvFixture(t *testing.T, exe string, input []byte, entries bool) []envPair {
	t.Helper()
	stdout, stderr, err := runEnvFixture(t, exe, input, entries)
	if err != nil {
		t.Fatalf("parse failed: %v\n%s", err, stderr)
	}
	if len(stderr) != 0 {
		t.Fatalf("stderr = %q", stderr)
	}
	return parseEnvDump(t, stdout)
}

func runEnvFixture(t *testing.T, exe string, input []byte, entries bool) ([]byte, []byte, error) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "fixture")
	if err := os.WriteFile(path, input, 0o644); err != nil {
		t.Fatal(err)
	}
	args := []string{path}
	if entries {
		args = append(args, "entries")
	}
	cmd := exec.Command(exe, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			err = fmt.Errorf("execute fixture: %w", err)
		} else {
			err = fmt.Errorf("execute fixture: %w\n%s", err, msg)
		}
	}
	return stdout.Bytes(), stderr.Bytes(), err
}

func parseEnvDump(t *testing.T, stdout []byte) []envPair {
	t.Helper()
	rest := stdout
	line, rest, ok := cutEnvLine(rest)
	if !ok {
		t.Fatalf("dump missing count: %q", stdout)
	}
	count, err := strconv.Atoi(line)
	if err != nil {
		t.Fatalf("count %q: %v", line, err)
	}
	entries := make([]envPair, 0, count)
	for i := 0; i < count; i++ {
		line, rest, ok = cutEnvLine(rest)
		if !ok {
			t.Fatalf("dump missing lengths at %d", i)
		}
		parts := strings.Fields(line)
		if len(parts) != 2 {
			t.Fatalf("lengths %q", line)
		}
		klen, err := strconv.Atoi(parts[0])
		if err != nil {
			t.Fatal(err)
		}
		vlen, err := strconv.Atoi(parts[1])
		if err != nil {
			t.Fatal(err)
		}
		if klen < 0 || vlen < 0 || len(rest) < klen+vlen {
			t.Fatalf("short dump at %d: key %d value %d remaining %d", i, klen, vlen, len(rest))
		}
		entries = append(entries, envPair{string(rest[:klen]), string(rest[klen : klen+vlen])})
		rest = rest[klen+vlen:]
	}
	if len(rest) != 0 {
		t.Fatalf("trailing dump bytes %q", rest)
	}
	return entries
}

func cutEnvLine(b []byte) (string, []byte, bool) {
	i := bytes.IndexByte(b, '\n')
	if i < 0 {
		return "", nil, false
	}
	line := b[:i]
	if len(line) > 0 && line[len(line)-1] == '\r' {
		line = line[:len(line)-1]
	}
	return string(line), b[i+1:], true
}

func assertEnvPairs(t *testing.T, got []envPair, want []envPair) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("entries:\n%s\nwant:\n%s", formatEnvPairs(got), formatEnvPairs(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("entry %d:\n%s\nwant:\n%s", i, formatEnvPairs(got), formatEnvPairs(want))
		}
	}
}

func appendU32(b []byte, v uint32) []byte {
	return append(b, byte(v), byte(v>>8), byte(v>>16), byte(v>>24))
}

func appendU64(b []byte, v uint64) []byte {
	var out []byte
	out = append(b, 0, 0, 0, 0, 0, 0, 0, 0)
	for i := 0; i < 8; i++ {
		out[len(b)+i] = byte(v >> (8 * i))
	}
	return out
}

func txmTensor(name string, dtype uint32, dims []uint64, payload []byte) []byte {
	var b []byte
	b = appendU32(b, uint32(len(name)))
	b = append(b, name...)
	b = appendU32(b, dtype)
	b = appendU32(b, uint32(len(dims)))
	for _, dim := range dims {
		b = appendU64(b, dim)
	}
	return append(b, payload...)
}

func txmFile(tensors ...[]byte) []byte {
	b := []byte{'T', 'X', 'M', '1'}
	b = appendU32(b, 1)
	b = appendU32(b, uint32(len(tensors)))
	for _, tensor := range tensors {
		b = append(b, tensor...)
	}
	return b
}

func cByteArray(name string, data []byte) string {
	var b strings.Builder
	fmt.Fprintf(&b, "static const uint8_t %s[] = {", name)
	for i, value := range data {
		if i > 0 {
			b.WriteByte(',')
		}
		fmt.Fprintf(&b, "%d", value)
	}
	b.WriteString("};")
	return b.String()
}

func TestTXMParseReadsTensors(t *testing.T) {
	floatOne := []byte{0, 0, 0, 0, 0, 0, 0xf0, 0x3f}
	ints := []byte{3, 0, 0, 0, 0, 0, 0, 0, 4, 0, 0, 0, 0, 0, 0, 0}
	good := txmFile(
		txmTensor("gamma", 2, []uint64{1}, floatOne),
		txmTensor("n", 1, []uint64{2}, ints),
	)
	compileAndRunRuntimeC(t, Source+cByteArray("good", good)+`
int main(void) {
    const char* err = rt_txm_parse(good, sizeof(good));
    if (err != NULL) {
        fprintf(stderr, "%s\n", err);
        return 1;
    }
    if (g_rt_txm.count != 2) return 2;
    if (g_rt_txm.tensors[0].dtype != 2 || g_rt_txm.tensors[0].rank != 1 || g_rt_txm.tensors[0].count != 1) return 3;
    if (g_rt_txm.tensors[1].dtype != 1 || g_rt_txm.tensors[1].count != 2) return 4;
    g_rt_txm.loaded = 1;
    rt_string gamma = `+cString("gamma")+`;
    rt_string n = `+cString("n")+`;
    if (rt_gpu_dim(gamma, 0) != 1) return 5;
    if (rt_gpu_scalar(gamma) != 1.0) return 6;
    rt_slice_float host = rt_gpu_host_floats(gamma);
    if (host.len != 1 || host.data[0] != 1.0) return 7;
    rt_slice_int counts = rt_gpu_host_ints(n);
    if (counts.len != 2 || counts.data[0] != 3 || counts.data[1] != 4) return 8;
    rt_txm_reset();
    if (g_rt_txm.count != 0) return 9;
    return 0;
}
`)
}

func TestTXMParseRejectsMalformedFiles(t *testing.T) {
	floatOne := []byte{0, 0, 0, 0, 0, 0, 0xf0, 0x3f}
	good := txmFile(txmTensor("gamma", 2, []uint64{1}, floatOne))
	badMagic := append([]byte{'X', 'X', 'X', 'X'}, good[4:]...)
	truncated := good[:len(good)-1]
	duplicate := txmFile(
		txmTensor("gamma", 2, []uint64{1}, floatOne),
		txmTensor("gamma", 2, []uint64{1}, floatOne),
	)
	overflow := txmFile(txmTensor("wide", 2, []uint64{1 << 62, 4}, nil))
	zeroDim := txmFile(txmTensor("empty", 2, []uint64{0}, nil))
	trailing := append(append([]byte{}, good...), 1)
	badVersion := append([]byte{}, good...)
	badVersion[4] = 2
	badRank := txmFile(txmTensor("rank", 2, []uint64{1, 1, 1, 1, 1}, floatOne))

	compileAndRunRuntimeC(t, Source+
		cByteArray("bad_magic", badMagic)+
		cByteArray("truncated", truncated)+
		cByteArray("duplicate", duplicate)+
		cByteArray("overflow", overflow)+
		cByteArray("zero_dim", zeroDim)+
		cByteArray("trailing", trailing)+
		cByteArray("bad_version", badVersion)+
		cByteArray("bad_rank", badRank)+`
static int expect_err(const uint8_t* data, size_t len, const char* want) {
    const char* err = rt_txm_parse(data, len);
    if (err == NULL || strcmp(err, want) != 0) {
        fprintf(stderr, "got %s\nwant %s\n", err == NULL ? "(none)" : err, want);
        return 0;
    }
    rt_txm_reset();
    return 1;
}

int main(void) {
    if (!expect_err(bad_magic, sizeof(bad_magic), "model file has a bad magic")) return 1;
    if (!expect_err(truncated, sizeof(truncated), "model file is truncated")) return 2;
    if (!expect_err(duplicate, sizeof(duplicate), "model file has a duplicate tensor name")) return 3;
    if (!expect_err(overflow, sizeof(overflow), "model tensor dimension product overflows")) return 4;
    if (!expect_err(zero_dim, sizeof(zero_dim), "model tensor dimension must be at least 1")) return 5;
    if (!expect_err(trailing, sizeof(trailing), "model file has trailing bytes")) return 6;
    if (!expect_err(bad_version, sizeof(bad_version), "model file has a bad version")) return 7;
    if (!expect_err(bad_rank, sizeof(bad_rank), "model tensor rank must be from 1 to 4")) return 8;
    return 0;
}
`)
}

func TestSampleModelFileParses(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "..", "examples", "projects", "digits_server", "sample.txm"))
	if err != nil {
		t.Fatal(err)
	}
	compileAndRunRuntimeC(t, Source+cByteArray("sample_txm", data)+`
int main(void) {
    const char* err = rt_txm_parse(sample_txm, sizeof(sample_txm));
    if (err != NULL) {
        fprintf(stderr, "%s\n", err);
        return 1;
    }
    if (g_rt_txm.count != 5) return 2;
    g_rt_txm.loaded = 1;
    rt_string support = `+cString("support")+`;
    rt_string dual = `+cString("dual")+`;
    rt_string intercept = `+cString("intercept")+`;
    rt_string n_support = `+cString("n_support")+`;
    rt_string gamma = `+cString("gamma")+`;
    if (rt_gpu_dim(support, 0) != 2 || rt_gpu_dim(support, 1) != 64) return 3;
    if (rt_gpu_dim(dual, 0) != 9 || rt_gpu_dim(dual, 1) != 2) return 4;
    if (rt_gpu_dim(intercept, 0) != 45) return 5;
    if (rt_gpu_dim(n_support, 0) != 10) return 6;
    if (!(rt_gpu_scalar(gamma) > 0.0009 && rt_gpu_scalar(gamma) < 0.0011)) return 7;
    rt_slice_int counts = rt_gpu_host_ints(n_support);
    if (counts.len != 10 || counts.data[0] != 1 || counts.data[1] != 1 || counts.data[2] != 0) return 8;
    rt_txm_reset();
    return 0;
}
`)
}

func TestGPUFreeRefusesProtectedPointer(t *testing.T) {
	compileAndRunRuntimeC(t, Source+`
int main(void) {
    int dummy = 0;
    int other = 0;
    rt_gpu_mark_owned(&dummy);
    if (!rt_gpu_is_owned(&dummy)) return 1;
    if (rt_gpu_is_owned(&other)) return 2;
    return 0;
}
`)
	compileAndRunRuntimeCExpectFailure(t, Source+`
int main(void) {
    int dummy = 0;
    rt_gpu_mark_owned(&dummy);
    rt_gpu_refuse_owned(&dummy);
    return 0;
}
`, "gpu.free cannot free a mounted tensor or workspace")
}

func TestParkedRequestArenaIsFreedOnce(t *testing.T) {
	compileAndRunRuntimeC(t, Source+`
int main(void) {
    rt_arena arena;
    rt_arena_init(&arena);
    rt_context ctx = {&arena, NULL};
    char* kept = (char*)rt_arena_alloc(&arena, 8);
    memcpy(kept, "parked!", 8);
    rt_http_park_slot slot;
    memset(&slot, 0, sizeof(slot));
    if (!rt_http_park_begin(&slot, &arena, &ctx)) return 1;
    if (arena.chunks != NULL) return 2;
    rt_arena_deinit(&arena);
    if (memcmp(kept, "parked!", 8) != 0) return 3;
    if (!rt_http_park_complete(&slot, 1)) return 4;
    if (slot.live || slot.completions != 1 || slot.sends != 1) return 5;
    if (rt_http_park_complete(&slot, 1)) return 6;
    if (slot.completions != 1 || slot.sends != 1) return 7;
    return 0;
}
`)
}

func formatEnvPairs(entries []envPair) string {
	var b strings.Builder
	for _, entry := range entries {
		fmt.Fprintf(&b, "%q=%q\n", entry.key, entry.value)
	}
	return b.String()
}

const envParseDriver = `
static void load_synthetic(rt_arena* arena) {
    const char* entries[] = {
        "PATH=from-proc",
        "BOTH=from-proc",
        "=C:=C:\\Users\\ada",
        "NOEQUALS",
        "=justvalue",
        "ProgramFiles(x86)=C:\\PF",
        "EMPTYPROC=",
        "EQ=a=b",
        "A B=kept",
        NULL
    };
    for (size_t i = 0; entries[i] != NULL; i++) {
        rt_env_add_entry(arena, entries[i], strlen(entries[i]));
    }
}

static int read_fixture(const char* path, uint8_t** out, size_t* out_len) {
    FILE* file = fopen(path, "rb");
    if (file == NULL) {
        return 1;
    }
    uint8_t* data = NULL;
    size_t len = 0;
    size_t cap = 0;
    for (;;) {
        if (len == cap) {
            size_t next = cap == 0 ? 256 : cap * 2;
            if (next < cap) {
                free(data);
                fclose(file);
                return 1;
            }
            uint8_t* grown = (uint8_t*)realloc(data, next);
            if (grown == NULL) {
                free(data);
                fclose(file);
                return 1;
            }
            data = grown;
            cap = next;
        }
        size_t n = fread(data + len, 1, cap - len, file);
        len += n;
        if (n == 0) {
            break;
        }
    }
    int failed = ferror(file) != 0;
    if (fclose(file) != 0) {
        failed = 1;
    }
    if (failed) {
        free(data);
        return 1;
    }
    *out = data;
    *out_len = len;
    return 0;
}

static int dump_env(void) {
    if (printf("%" PRId64 "\n", rt_env_len()) < 0) {
        return 1;
    }
    for (int64_t i = 0; i < rt_env_len(); i++) {
        rt_string key = rt_env_key(i);
        rt_string value = rt_env_value(i);
        if (printf("%" PRId64 " %" PRId64 "\n", (int64_t)key.len, (int64_t)value.len) < 0) {
            return 1;
        }
        if (key.len > 0 && fwrite(key.data, 1, key.len, stdout) != key.len) {
            return 1;
        }
        if (value.len > 0 && fwrite(value.data, 1, value.len, stdout) != value.len) {
            return 1;
        }
    }
    return 0;
}

int main(int argc, char** argv) {
    if (argc < 2) {
        fprintf(stderr, "missing fixture\n");
        return 2;
    }
    int with_entries = 0;
    if (argc > 2) {
        if (strcmp(argv[2], "entries") != 0) {
            fprintf(stderr, "unknown mode\n");
            return 2;
        }
        with_entries = 1;
    }
    rt_arena arena;
    rt_arena_init(&arena);
    if (with_entries) {
        load_synthetic(&arena);
    }
    uint8_t* data = NULL;
    size_t len = 0;
    if (read_fixture(argv[1], &data, &len) != 0) {
        fprintf(stderr, "cannot read fixture\n");
        return 1;
    }
    if (len > 0) {
        rt_env_parse(&arena, data, len);
    }
    free(data);
    if (dump_env() != 0) {
        fprintf(stderr, "dump failed\n");
        return 1;
    }
    rt_arena_deinit(&arena);
    return 0;
}
`
