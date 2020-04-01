package builder

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/tinygo-org/tinygo/compileopts"
	"github.com/tinygo-org/tinygo/goenv"
)

// Library is a container for information about a single C library, such as a
// compiler runtime or libc.
type Library struct {
	// The library name, such as compiler-rt or picolibc.
	name string

	// makeHeaders creates a header include dir for the library
	makeHeaders func(target, includeDir string) error

	// cflags returns the C flags specific to this library
	cflags func(target, outTempDir string) []string

	// The source directory, relative to TINYGOROOT.
	sourceDir string

	// The source files, relative to sourceDir.
	librarySources func(target string) []string

	// The source code for the crt1.o file, relative to sourceDir.
	crt1Source string
}

// fullPath returns the full path to the source directory.
func (l *Library) fullPath() string {
	return filepath.Join(goenv.Get("TINYGOROOT"), l.sourceDir)
}

// sourcePaths returns a slice with the full paths to the source (library and
// crt1) files.
func (l *Library) sourcePaths(target string) []string {
	sources := l.librarySources(target)
	if l.crt1Source != "" {
		sources = append([]string{l.crt1Source}, sources...)
	}
	paths := make([]string, len(sources))
	for i, name := range sources {
		paths[i] = filepath.Join(l.fullPath(), name)
	}
	return paths
}

// Load the library archive, possibly generating and caching it if needed.
// The resulting directory may be stored in the provided tmpdir, which is
// expected to be removed after the Load call.
func (l *Library) Load(config *compileopts.Config, tmpdir string) (dir string, err error) {
	job, err := l.load(config, tmpdir)
	if err != nil {
		return "", err
	}
	err = runJobs(job)
	return filepath.Dir(job.result), err
}

// load returns a compile job to build this library file for the given target
// and CPU. It may return a dummy compileJob if the library build is already
// cached. The path is stored as job.result but is only valid if the job has
// been run.
// The provided tmpdir will be used to store intermediary files and possibly the
// output archive file, it is expected to be removed after use.
func (l *Library) load(config *compileopts.Config, tmpdir string) (job *compileJob, err error) {
	outdir, precompiled := config.LibcPath(l.name)
	if precompiled {
		// Found a precompiled library for this OS/architecture. Return the path
		// directly.
		return dummyCompileJob(outdir), nil
	}

	// Try to fetch this library from the cache.
	outname := filepath.Base(outdir)
	target := config.Triple()
	if path, err := cacheLoad(outname, l.sourcePaths(target)); path != "" || err != nil {
		// Cache hit.
		return dummyCompileJob(filepath.Join(path, "lib.a")), nil
	}
	// Cache miss, build it now.

	// Temporary directory (inside the cache directory) where the library is
	// created. It is later moved to the final location (without the .tmp1234
	// suffix).
	outtmpdir, err := ioutil.TempDir(goenv.Get("GOCACHE"), outname+".tmp*")
	if err != nil {
		return nil, err
	}

	remapDir := filepath.Join(os.TempDir(), "tinygo-"+l.name)
	dir := filepath.Join(tmpdir, "build-lib-"+l.name)
	err = os.Mkdir(dir, 0777)
	if err != nil {
		return nil, err
	}

	// Precalculate the flags to the compiler invocation.
	// Note: -fdebug-prefix-map is necessary to make the output archive
	// reproducible. Otherwise the temporary directory is stored in the archive
	// itself, which varies each run.
	args := append(l.cflags(target, outtmpdir), "-c", "-Oz", "-g", "-ffunction-sections", "-fdata-sections", "-Wno-macro-redefined", "--target="+target, "-fdebug-prefix-map="+dir+"="+remapDir)
	cpu := config.CPU()
	if cpu != "" {
		args = append(args, "-mcpu="+cpu)
	}
	if strings.HasPrefix(target, "arm") || strings.HasPrefix(target, "thumb") {
		args = append(args, "-fshort-enums", "-fomit-frame-pointer", "-mfloat-abi=soft")
	}
	if strings.HasPrefix(target, "riscv32-") {
		args = append(args, "-march=rv32imac", "-mabi=ilp32", "-fforce-enable-int128")
	}
	if strings.HasPrefix(target, "riscv64-") {
		args = append(args, "-march=rv64gc", "-mabi=lp64")
	}

	// Create job to put all the object files in a single archive. This archive
	// file is the (static) library file.
	var objs []string
	job = &compileJob{
		description: "ar " + l.name + "/lib.a",
		result:      filepath.Join(goenv.Get("GOCACHE"), outname, "lib.a"),
		run: func(*compileJob) error {
			// Create an archive of all object files.
			err := makeArchive(filepath.Join(outtmpdir, "lib.a"), objs)
			if err != nil {
				return fmt.Errorf("failed to make archive for %s: %w", target, err)
			}
			// Store this archive in the cache.
			_, err = cacheStore(outtmpdir, outname, l.sourcePaths(target))
			return err
		},
	}

	// Create header files if needed.
	var compileDependencies []*compileJob
	includeDir := filepath.Join(outtmpdir, "include")
	if l.makeHeaders != nil {
		compileDependencies = append(compileDependencies, &compileJob{
			description: "headers " + l.name + "/include",
			run: func(*compileJob) error {
				err := os.Mkdir(includeDir, 0777)
				if err != nil {
					return err
				}
				return l.makeHeaders(target, includeDir)
			},
		})
	}

	// Create jobs to compile all sources. These jobs are depended upon by the
	// archive job above, so must be run first.
	for _, path := range l.librarySources(target) {
		srcpath := filepath.Join(l.sourceDir, path)
		objpath := filepath.Join(dir, filepath.Base(srcpath)+".o")
		objs = append(objs, objpath)
		job.dependencies = append(job.dependencies, &compileJob{
			description:  "compile " + srcpath,
			dependencies: compileDependencies,
			run: func(*compileJob) error {
				var compileArgs []string
				compileArgs = append(compileArgs, args...)
				compileArgs = append(compileArgs, "-o", objpath, srcpath)
				err := runCCompiler(compileArgs...)
				if err != nil {
					return &commandError{"failed to build", srcpath, err}
				}
				return nil
			},
		})
	}

	// Create crt1.o job, if needed.
	// Add this as a (fake) dependency to the ar file so it gets compiled.
	// (It could be done in parallel with creating the ar file, but it probably
	// won't make much of a difference in speed).
	if l.crt1Source != "" {
		srcpath := filepath.Join(l.sourceDir, l.crt1Source)
		objpath := filepath.Join(outtmpdir, "crt1.o")
		job.dependencies = append(job.dependencies, &compileJob{
			description:  "compile " + srcpath,
			dependencies: compileDependencies,
			run: func(*compileJob) error {
				var compileArgs []string
				compileArgs = append(compileArgs, args...)
				compileArgs = append(compileArgs, "-o", objpath, srcpath)
				err := runCCompiler(compileArgs...)
				if err != nil {
					return &commandError{"failed to build", srcpath, err}
				}
				return nil
			},
		})
	}

	return job, nil
}
