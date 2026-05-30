package c

import (
	"os"
	"os/exec"
	"path/filepath"
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
