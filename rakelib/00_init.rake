# coding: utf-8

# Initialization
# This file contains the toolkits that
# aren't related to the source code.
# It means that they don't change very often
# and can be cached for later use.

require 'open3'

# Cross-platform way of finding an executable in the $PATH.
# Source: https://stackoverflow.com/a/5471032
#
#   which('ruby') #=> /usr/bin/ruby
def which(cmd)
    if File.executable?(cmd)
        return cmd
    end

    exts = ENV['PATHEXT'] ? ENV['PATHEXT'].split(';') : ['']
    ENV['PATH'].split(File::PATH_SEPARATOR).each do |path|
      exts.each do |ext|
        exe = File.join(path, "#{cmd}#{ext}")
        return exe if File.executable?(exe) && !File.directory?(exe)
      end
    end
    nil
end

# Returns true if the libc-musl variant of the libc library is used. Otherwise,
# returns false (the standard variant is used).
def detect_libc_musl()
    platform = Gem::Platform.local
    if platform.version.nil?
        return false
    end
    return platform.version == "musl"
end

# Searches for the tasks in the provided file
def find_tasks(file)
    tasks = []
    # Iterate over all tasks
    Rake.application.tasks.each do |t|
        # Choose only tasks from a specific file
        if t.actions.empty?
            next
        end
        action = t.actions[0]
        location, _ = action.source_location
        if location != file
            next
        end
        tasks.append t
    end
    return tasks
end

# Searches for the prerequisites tasks from the second provided file in the
# first provided file.
def find_prerequisites_tasks(source_tasks_file, prerequisites_file)
    require 'set'
    unique_prerequisites = Set[]

    # Choose only tasks from a specific file
    tasks = find_tasks(source_tasks_file)

    # Iterate over tasks
    tasks.each do |t|
        # Iterate over prerequisites
        t.all_prerequisite_tasks.each do |p|
            # Select unique prerequisites
            unique_prerequisites.add p
        end
    end

    prerequisites_tasks = []

    # Check the prerequisites
    unique_prerequisites.each do |p|
        # Check the location - accept only tasks from the init file
        if p.actions.empty?
            next
        end

        action = p.actions[0]
        location, _ = action.source_location

        if location == prerequisites_file
            prerequisites_tasks.append p
        end
    end

    return prerequisites_tasks
end

# Searches for the prerequisites from the init file in the provided file and
# invoke them.
def find_and_prepare_deps(file)
    if file == __FILE__
        prerequisites_tasks = find_tasks(file)
    else
        prerequisites_tasks = find_prerequisites_tasks(file, __FILE__)
    end

    prerequisites_tasks.each do |t|
        if t.class != Rake::FileTask
            next
        end
        print "Preparing: ", t, "...\n"
        t.invoke()
    end
end

# Searches for the prerequisites from the init file in the provided file and
# checks if they exist. It accepts the system-wide dependencies list and tests
# if they are in PATH.
def check_deps(file, *system_deps)
    def print_status(name, path, ok)
        status = "[ OK ]"
        if !ok
            status = "[MISS]"
        end

        if !path.nil?
            path = " (" + path + ")"
        end

        print status, " ", name, path, "\n"
    end

    if file == __FILE__
        prerequisites_tasks = find_tasks(file)
    else
        prerequisites_tasks = find_prerequisites_tasks(file, __FILE__)
    end

    manual_install_prerequisites_tasks = []
    prerequisites_tasks.each do |t|
        if t.instance_variable_get(:@manuall_install)
            manual_install_prerequisites_tasks.append t
        end
    end

    manual_install_prerequisites_tasks.each do |t|
        prerequisites_tasks.delete t
    end

    puts "Prerequisites:"
    prerequisites_tasks.sort_by{ |t| t.to_s().rpartition("/")[2] }.each do |t|
        if t.class != Rake::FileTask
            next
        end

        path = t.to_s
        name = path
        _, _, name = path.rpartition("/")

        print_status(name, path, File.exist?(path))
    end

    puts "System dependencies:"

    system_deps
        .map { |d| [d.to_s, which(d)] }
        .map { |d, path| [d, path, !path.nil?] }
        .concat(
            manual_install_prerequisites_tasks
            .map { |p| [p.to_s().rpartition("/")[2], p.to_s() ] }
            .map { |p, path| [p, path, File.exist?(path)] }
        )
        .sort_by{ |name, _, _| name }
        .each { |args| print_status(*args) }
end

# Defines the version guard for a file task. The version guard allows file
# tasks to depend on the version from the Rake variable. Using it for the tasks
# that have frozen versions using external files is not necessary.
# It accepts a task to be guarded and the version.
def add_version_guard(task_name, version)
    task = Rake::Task[task_name]
    if task.class != Rake::FileTask
        fail "file task required"
    end

    # We don't use the version guard for the prerequisities that must be
    # installed manually on current operating system
    if task.instance_variable_get(:@manuall_install)
        return
    end

    # The version stamp file is a prerequisite, but it is created after the
    # guarded task. It allows for cleaning the target directory in the task
    # body.
    version_stamp = "#{task_name}-#{version}.version"
    file version_stamp
    task.enhance [version_stamp] do
        # Removes old version stamps
        FileList["#{task_name}-*.version"].each do |f|
            FileUtils.rm f
        end
        # Creates a new version stamp with a timestamp before the guarded task
        # execution.
        FileUtils.touch [version_stamp], mtime: task.timestamp
    end 
end


def require_manual_install_on(task_name, *conditions)
    task = Rake::Task[task_name]
    if task.class != Rake::FileTask
        fail "file task required"
    end

    if !conditions.any?
        return
    end

    task.instance_variable_set(:@manuall_install, true)
    task.clear_actions()
    task.clear_prerequisites()
    task.enhance do
        fail "#{task.to_s} must be installed manually on your operating system"
    end
end

### Recognize the operating system
uname=`uname -s`

case uname.rstrip
    when "Darwin"
        OS="macos"
    when "Linux"
        OS="linux"
    when "FreeBSD"
        OS="FreeBSD"
    when "OpenBSD"
        OS="OpenBSD"    
    else
        puts "ERROR: Unknown/unsupported OS: %s" % UNAME
        fail
end

### Tasks support conditions
# Some prerequisites are related to the libc library but
# without official libc-musl variants. They cannot be installed using this Rake
# script.
libc_musl_system = detect_libc_musl()
# Some prerequisites doesn't have a public packages for BSD-like operating
# systems.
freebsd_system = OS == "FreeBSD"
openbsd_system = OS == "OpenBSD"

### Detect wget
if which("wget").nil?
    abort("wget is not installed on this system")
end
# extract wget version
stdout, _, status = Open3.capture3("wget", "--version")
wget = ["wget"]

# BusyBox edition has no version switch and supports only basic features.
if status == 0
    wget.append "--tries=inf", "--waitretry=3"
    wget_version = stdout.split("\n")[0]
    wget_version = wget_version[/[0-9]+\.[0-9]+/]
    # versions prior to 1.19 lack support for --retry-on-http-error
    if wget_version.empty? or wget_version >= "1.19"
        wget.append "--retry-on-http-error=429,500,503,504"
    end
end

if ENV["CI"] == "true"
    wget.append "-q"
end
WGET = wget

### Define package versions
go_ver='1.18.3'
openapi_generator_ver='5.2.0'
goswagger_ver='v0.23.0'
protoc_ver='3.18.1'
protoc_gen_go_ver='v1.26.0'
protoc_gen_go_grpc_ver='v1.1.0'
richgo_ver='v0.3.10'
mockery_ver='v2.13.1'
mockgen_ver='v1.6.0'
golangcilint_ver='v1.46.2'
yamlinc_ver='0.1.10'
node_ver='14.18.2'
dlv_ver='v1.8.3'
gdlv_ver='v1.8.0'
bundler_ver='2.3.8'
storybook_ver='6.5.10'

# System-dependent variables
case OS
when "macos"
  go_suffix="darwin-amd64"
  protoc_suffix="osx-x86_64"
  node_suffix="darwin-x64"
  golangcilint_suffix="darwin-amd64"
  chrome_drv_suffix="mac64"
  puts "WARNING: MacOS is not officially supported, the provisions for building on MacOS are made"
  puts "WARNING: for the developers' convenience only."
when "linux"
  go_suffix="linux-amd64"
  protoc_suffix="linux-x86_64"
  node_suffix="linux-x64"
  golangcilint_suffix="linux-amd64"
  chrome_drv_suffix="linux64"
when "FreeBSD"
  go_suffix="freebsd-amd64"
  golangcilint_suffix="freebsd-amd64"
when "OpenBSD"
else
  puts "ERROR: Unknown/unsupported OS: %s" % UNAME
  fail
end

### Detect Chrome
# CHROME_BIN is required for UI unit tests and system tests. If it is
# not provided by a user, try to locate Chrome binary and set
# environment variable to its location.
if !ENV['CHROME_BIN'] || ENV['CHROME_BIN'].empty?
    location = which("chromium")
    if location.nil?
        location = which("chrome")
    end
    if !location.nil?
        ENV['CHROME_BIN'] = location
    else    
        ENV['CHROME_BIN'] = "chromium"
        chrome_locations = []

        if OS == 'linux'
            chrome_locations = ['/usr/bin/chromium-browser', '/snap/bin/chromium', '/usr/bin/chromium']
        elsif OS == 'macos'
            chrome_locations = ["/Applications/Google\ Chrome.app/Contents/MacOS/Google\ Chrome"]
        end
        # For each possible location check if the binary exists.
        chrome_locations.each do |loc|
            if File.exist?(loc)
                # Found Chrome binary.
                ENV['CHROME_BIN'] = loc
                break
            end
        end
    end
end

### Define dependencies

# Directories
tools_dir = File.expand_path('tools')
directory tools_dir

node_dir = File.join(tools_dir, "nodejs")
directory node_dir

go_tools_dir = File.join(tools_dir, "golang")
gopath = File.join(go_tools_dir, "gopath")
directory go_tools_dir
directory gopath
file go_tools_dir => [gopath]

ruby_tools_dir = File.join(tools_dir, "ruby")
directory ruby_tools_dir

# We use the "bundle" gem to manage the dependencies. The "bundle" package is
# installed using the "gem" executable in the tools/ruby/gems directory, and 
# the link is created in the tools/ruby/bin directory. Next, Ruby dependencies
# are installed using the "bundle". It creates the tools/ruby/ruby/[VERSION]/
# directory with "bin" and "gems" subdirectories and uses these directories as
# the location of the installations. We want to avoid using a variadic Ruby
# version in the directory name. Therefore, we use the "binstubs" feature to
# create the links to the executable. Unfortunately, if we use the
# "tools/ruby/bin" directory as the target location then the "bundle"
# executable will be overridden and stop working. To work around this problem,
# we use two directories for Ruby binaries. The first contains the binaries
# installed using the "gem" command, and the second is a target for the
# "bundle" command.
ruby_tools_bin_dir = File.join(ruby_tools_dir, "bin")
directory ruby_tools_bin_dir
ruby_tools_bin_bundle_dir = File.join(ruby_tools_dir, "bin_bundle")
directory ruby_tools_bin_bundle_dir

# Automatically created directories by tools
ruby_tools_gems_dir = File.join(ruby_tools_dir, "gems")
goroot = File.join(go_tools_dir, "go")
gobin = File.join(goroot, "bin")
python_tools_dir = File.join(tools_dir, "python")
pythonpath = File.join(python_tools_dir, "lib")
node_bin_dir = File.join(node_dir, "bin")
protoc_dir = go_tools_dir

default_os_binary_directory = "/usr/bin"
if freebsd_system || openbsd_system
    default_os_binary_directory = "/usr/local/bin"
end
DEFAULT_OS_BINARY_DIRECTORY = default_os_binary_directory

if libc_musl_system || freebsd_system || openbsd_system
    protoc_dir = DEFAULT_OS_BINARY_DIRECTORY
    node_bin_dir = DEFAULT_OS_BINARY_DIRECTORY
end

if libc_musl_system || openbsd_system
    gobin = ENV["GOBIN"]
    goroot = ENV["GOROOT"]
    if gobin.nil?
        gobin = which("go")
        if !gobin.nil?
            gobin = File.dirname gobin
        else
            gobin = DEFAULT_OS_BINARY_DIRECTORY
        end
    end
end

# Environment variables
ENV["GEM_HOME"] = ruby_tools_dir
ENV["BUNDLE_PATH"] = ruby_tools_dir
ENV["BUNDLE_BIN"] = ruby_tools_bin_bundle_dir
ENV["GOROOT"] = goroot
ENV["GOPATH"] = gopath
ENV["GOBIN"] = gobin
ENV["PATH"] = "#{node_bin_dir}:#{tools_dir}:#{gobin}:#{ENV["PATH"]}"
ENV["PYTHONPATH"] = pythonpath
ENV["VIRTUAL_ENV"] = python_tools_dir

# Toolkits
BUNDLE = File.join(ruby_tools_bin_dir, "bundle")
file BUNDLE => [ruby_tools_dir, ruby_tools_bin_dir] do
    sh "gem", "install",
            "--minimal-deps",
            "--no-document",
            "--install-dir", ruby_tools_dir,
            "bundler:#{bundler_ver}"

    if !File.exists? BUNDLE
        # Workaround for old Ruby versions
        sh "ln", "-s", File.join(ruby_tools_gems_dir, "bundler-#{bundler_ver}", "exe", "bundler"), File.join(ruby_tools_bin_dir, "bundler")
        sh "ln", "-s", File.join(ruby_tools_gems_dir, "bundler-#{bundler_ver}", "exe", "bundle"), BUNDLE
    end

    sh BUNDLE, "--version"
end
add_version_guard(BUNDLE, bundler_ver)

fpm_gemfile = File.expand_path("init_deps/fpm.Gemfile", __dir__)
FPM = File.join(ruby_tools_bin_bundle_dir, "fpm")
file FPM => [BUNDLE, ruby_tools_dir, ruby_tools_bin_bundle_dir, fpm_gemfile] do
    sh BUNDLE, "install",
        "--gemfile", fpm_gemfile,
        "--path", ruby_tools_dir,
        "--binstubs", ruby_tools_bin_bundle_dir
    sh FPM, "--version"
end

danger_gemfile = File.expand_path("init_deps/danger.Gemfile", __dir__)
DANGER = File.join(ruby_tools_bin_bundle_dir, "danger")
file DANGER => [ruby_tools_bin_bundle_dir, ruby_tools_dir, danger_gemfile, BUNDLE] do
    sh BUNDLE, "install",
        "--gemfile", danger_gemfile,
        "--path", ruby_tools_dir,
        "--binstubs", ruby_tools_bin_bundle_dir
    sh "touch", "-c", DANGER
    sh DANGER, "--version"
end

NPM = File.join(node_bin_dir, "npm")
file NPM => [node_dir] do
    Dir.chdir(node_dir) do
        FileUtils.rm_rf("*")
        sh *WGET, "https://nodejs.org/dist/v#{node_ver}/node-v#{node_ver}-#{node_suffix}.tar.xz", "-O", "node.tar.xz"
        sh "tar", "-Jxf", "node.tar.xz", "--strip-components=1"
        sh "rm", "node.tar.xz"
    end
    sh NPM, "--version"
end
require_manual_install_on(NPM, libc_musl_system, freebsd_system, openbsd_system)
add_version_guard(NPM, node_ver)

NPX = File.join(node_bin_dir, "npx")
file NPX => [NPM] do
    sh NPX, "--version"
    sh "touch", "-c", NPX
end
require_manual_install_on(NPX, libc_musl_system)

YAMLINC = File.join(node_dir, "node_modules", "lib", "node_modules", "yamlinc", "bin", "yamlinc")
file YAMLINC => [NPM] do
    ci_opts = []
    if ENV["CI"] == "true"
        ci_opts += ["--no-audit", "--no-progress"]
    end

    sh NPM, "install",
            "-g",
            *ci_opts,
            "--prefix", "#{node_dir}/node_modules",
            "yamlinc@#{yamlinc_ver}"
    sh "touch", "-c", YAMLINC
    sh YAMLINC, "--version"
end
add_version_guard(YAMLINC, yamlinc_ver)

STORYBOOK = File.join(node_dir, "node_modules", "bin", "sb")
file STORYBOOK => [NPM] do
    ci_opts = []
    if ENV["CI"] == "true"
        ci_opts += ["--no-audit", "--no-progress"]
    end

    sh NPM, "install",
            "-g",
            *ci_opts,
            "--prefix", "#{node_dir}/node_modules",
            "storybook@#{storybook_ver}"
    sh "touch", "-c", STORYBOOK
    sh STORYBOOK, "--version"
end
add_version_guard(STORYBOOK, storybook_ver)

# Chrome driver is not currently used, but it can be needed in the UI tests.
# This file task is ready to use after uncomment.
#
# puts "WARNING: There are no chrome drv packages built for FreeBSD"
#
# CHROME_DRV = File.join(tools_dir, "chromedriver")
# file CHROME_DRV => [tools_dir] do
#     if !ENV['CHROME_BIN']
#         puts "Missing Chrome/Chromium binary. It is required for UI unit tests and system tests."
#         next
#     end

#     chrome_version = `"#{ENV['CHROME_BIN']}" --version | cut -d" " -f2 | tr -d -c 0-9.`
#     chrome_drv_version = chrome_version

#     if chrome_version.include? '85.'
#         chrome_drv_version = '85.0.4183.87'
#     elsif chrome_version.include? '86.'
#         chrome_drv_version = '86.0.4240.22'
#     elsif chrome_version.include? '87.'
#         chrome_drv_version = '87.0.4280.20'
#     elsif chrome_version.include? '90.'
#         chrome_drv_version = '90.0.4430.72'
#     elsif chrome_version.include? '92.'
#         chrome_drv_version = '92.0.4515.159'
#     elsif chrome_version.include? '93.'
#         chrome_drv_version = '93.0.4577.63'
#     elsif chrome_version.include? '94.'
#         chrome_drv_version = '94.0.4606.61' 
#     end

#     Dir.chdir(tools_dir) do
#         sh *WGET, "https://chromedriver.storage.googleapis.com/#{chrome_drv_version}/chromedriver_#{chrome_drv_suffix}.zip", "-O", "chromedriver.zip"
#         sh "unzip", "-o", "chromedriver.zip"
#         sh "rm", "chromedriver.zip"
#     end

#     sh CHROME_DRV, "--version"
#     sh "chromedriver", "--version"  # From PATH
# end

OPENAPI_GENERATOR = File.join(tools_dir, "openapi-generator-cli.jar")
file OPENAPI_GENERATOR => [tools_dir] do
    sh *WGET, "https://repo1.maven.org/maven2/org/openapitools/openapi-generator-cli/#{openapi_generator_ver}/openapi-generator-cli-#{openapi_generator_ver}.jar", "-O", OPENAPI_GENERATOR
end
add_version_guard(OPENAPI_GENERATOR, openapi_generator_ver)

GO = File.join(gobin, "go")
file GO => [go_tools_dir] do
    Dir.chdir(go_tools_dir) do
        FileUtils.rm_rf("go")
        sh *WGET, "https://dl.google.com/go/go#{go_ver}.#{go_suffix}.tar.gz", "-O", "go.tar.gz"
        sh "tar", "-zxf", "go.tar.gz" 
        sh "rm", "go.tar.gz"
    end
    sh "touch", "-c", GO
    sh GO, "version"
end
require_manual_install_on(GO, libc_musl_system, openbsd_system)
add_version_guard(GO, go_ver)

GOSWAGGER = File.join(go_tools_dir, "goswagger")
file GOSWAGGER => [GO, go_tools_dir] do
    if OS != 'FreeBSD' && OS != "OpenBSD"
        goswagger_suffix = "linux_amd64"
        if OS == 'macos'
            # GoSwagger fails to build on macOS due to https://gitlab.isc.org/isc-projects/stork/-/issues/848.
            goswagger_suffix="darwin_amd64"
        end
        sh *WGET, "https://github.com/go-swagger/go-swagger/releases/download/#{goswagger_ver}/swagger_#{goswagger_suffix}", "-O", GOSWAGGER
        sh "chmod", "u+x", GOSWAGGER
    else
        # GoSwagger lacks the packages for BSD-like systems then it must be
        # built from sources.
        goswagger_archive = "#{GOSWAGGER}.tar.gz"
        goswagger_dir = "#{GOSWAGGER}-sources"
        sh "mkdir", goswagger_dir
        sh *WGET, "https://github.com/go-swagger/go-swagger/archive/refs/tags/#{goswagger_ver}.tar.gz", "-O", goswagger_archive
        sh "tar", "-zxf", goswagger_archive, "-C", goswagger_dir, "--strip-components=1"
        goswagger_build_dir = File.join(goswagger_dir, "cmd", "swagger")
        Dir.chdir(goswagger_build_dir) do
            sh GO, "build", "-ldflags=-X 'github.com/go-swagger/go-swagger/cmd/swagger/commands.Version=#{goswagger_ver}'"
        end
        sh "mv", File.join(goswagger_build_dir, "swagger"), GOSWAGGER
        sh "rm", "-rf", goswagger_dir
        sh "rm", goswagger_archive
    end

    sh "touch", "-c", GOSWAGGER
    sh GOSWAGGER, "version"
end
add_version_guard(GOSWAGGER, goswagger_ver)

PROTOC = File.join(protoc_dir, "protoc")
file PROTOC => [go_tools_dir] do
    Dir.chdir(go_tools_dir) do
        sh *WGET, "https://github.com/protocolbuffers/protobuf/releases/download/v#{protoc_ver}/protoc-#{protoc_ver}-#{protoc_suffix}.zip", "-O", "protoc.zip"
        sh "unzip", "-o", "-j", "protoc.zip", "bin/protoc"
        sh "rm", "protoc.zip"
    end
    sh PROTOC, "--version"
    sh "touch", "-c", PROTOC
end
require_manual_install_on(PROTOC, libc_musl_system, freebsd_system, openbsd_system)
add_version_guard(PROTOC, protoc_ver)

PROTOC_GEN_GO = File.join(gobin, "protoc-gen-go")
file PROTOC_GEN_GO => [GO] do
    sh GO, "install", "google.golang.org/protobuf/cmd/protoc-gen-go@#{protoc_gen_go_ver}"
    sh PROTOC_GEN_GO, "--version"
end
add_version_guard(PROTOC_GEN_GO, protoc_gen_go_ver)

PROTOC_GEN_GO_GRPC = File.join(gobin, "protoc-gen-go-grpc")
file PROTOC_GEN_GO_GRPC => [GO] do
    sh GO, "install", "google.golang.org/grpc/cmd/protoc-gen-go-grpc@#{protoc_gen_go_grpc_ver}"
    sh PROTOC_GEN_GO_GRPC, "--version"
end
add_version_guard(PROTOC_GEN_GO_GRPC, protoc_gen_go_grpc_ver)

GOLANGCILINT = File.join(gobin, "golangci-lint")
file GOLANGCILINT => [go_tools_dir] do
    sh GO, "install", "github.com/golangci/golangci-lint/cmd/golangci-lint@#{golangcilint_ver}"
    sh GOLANGCILINT, "--version"
end
require_manual_install_on(GOLANGCILINT, openbsd_system)
add_version_guard(GOLANGCILINT, golangcilint_ver)

RICHGO = "#{gobin}/richgo"
file RICHGO => [GO] do
    sh GO, "install", "github.com/kyoh86/richgo@#{richgo_ver}"
    sh RICHGO, "version"
end
add_version_guard(RICHGO, richgo_ver)

MOCKERY = "#{gobin}/mockery"
file MOCKERY => [GO] do
    sh GO, "install", "github.com/vektra/mockery/v2@#{mockery_ver}"
    sh MOCKERY, "--version"
end
add_version_guard(MOCKERY, mockery_ver)

MOCKGEN = "#{gobin}/mockgen"
file MOCKGEN => [GO] do
    sh GO, "install", "github.com/golang/mock/mockgen@#{mockgen_ver}"
    sh MOCKGEN, "--version"
end
add_version_guard(MOCKGEN, mockgen_ver)

DLV = "#{gobin}/dlv"
file DLV => [GO] do
    sh GO, "install", "github.com/go-delve/delve/cmd/dlv@#{dlv_ver}"
    sh DLV, "version"
end
add_version_guard(DLV, dlv_ver)

GDLV = "#{gobin}/gdlv"
file GDLV => [GO] do
    sh GO, "install", "github.com/aarzilli/gdlv@#{gdlv_ver}"
    if !File.file?(GDLV)
        fail
    end
end
add_version_guard(GDLV, gdlv_ver)

PYTHON = File.join(python_tools_dir, "bin", "python")
file PYTHON do
    sh "python3", "-m", "venv", python_tools_dir
    sh PYTHON, "--version"
end

PIP = File.join(python_tools_dir, "bin", "pip")
file PIP => [PYTHON] do
    sh PYTHON, "-m", "ensurepip", "-U", "--default-pip"
    sh "touch", "-c", PIP
    sh PIP, "--version"
end

SPHINX_BUILD = File.expand_path("tools/python/bin/sphinx-build")
sphinx_requirements_file = File.expand_path("init_deps/sphinx.txt", __dir__)
file SPHINX_BUILD => [PIP, sphinx_requirements_file] do
    sh PIP, "install", "-r", sphinx_requirements_file
    sh "touch", "-c", SPHINX_BUILD
    sh SPHINX_BUILD, "--version"
end

PYTEST = File.expand_path("tools/python/bin/pytest")
pytests_requirements_file = File.expand_path("init_deps/pytest.txt", __dir__)
file PYTEST => [PIP, pytests_requirements_file] do
    sh PIP, "install", "-r", pytests_requirements_file
    sh "touch", "-c", PYTEST
    sh PYTEST, "--version"
end

#############
### Tasks ###
#############

desc 'Install all system-level dependencies'
task :prepare do
    find_and_prepare_deps(__FILE__)
end

desc 'Check all system-level dependencies'
task :check do
    system_specific_deps = []
    if OS == "OpenBSD"
        system_specific_deps.append "clang++"
    end
    check_deps(__FILE__, "wget", "python3", "java", "unzip", "entr", "git",
        "createdb", "psql", "dropdb", ENV['CHROME_BIN'], "docker-compose",
        "docker", "openssl", "gem", "make", "gcc", "tar", "sed", "perl", "fold",
        "ssh", "scp", *system_specific_deps)
end
