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
    puts "Prerequisites:"
    
    if file == __FILE__
        prerequisites_tasks = find_tasks(file)
    else
        prerequisites_tasks = find_prerequisites_tasks(file, __FILE__)
    end

    prerequisites_tasks.sort_by{ |t| t.to_s().rpartition("/")[2] }.each do |t|
        if t.class != Rake::FileTask
            next
        end

        path = t.to_s
        name = path
        _, _, name = path.rpartition("/")

        status = "[ OK ]"
        if !File.exist?(path)
            status = "[MISS]"
        end

        print status, " ", name, " (", path, ")\n"

    end

    puts "System dependencies:"

    system_deps.sort.each do |d|
        status = "[ OK ]"
        path = which(d)
        if path.nil?
            status = "[MISS]"
        else
            path = " (" + path + ")"
        end
        print status, " ", d, path, "\n"
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
  else
    puts "ERROR: Unknown/unsupported OS: %s" % UNAME
    fail
  end

### Detect wget
if system("wget --version > /dev/null").nil?
    abort("wget is not installed on this system")
end
# extract wget version
wget_version = `wget --version | head -n 1 | sed -E 's/[^0-9]*([0-9]*\.[0-9]*)[^0-9]+.*/\1/g'`
# versions prior to 1.19 lack support for --retry-on-http-error
wget = ["wget", "--tries=inf", "--waitretry=3"]
if wget_version.empty? or wget_version >= "1.19"
    wget = wget + ["--retry-on-http-error=429,500,503,504"]
end

if ENV["CI"] == "true"
    wget = wget + ["-q"]
end
WGET = wget

### Define package versions
go_ver='1.17.5'
openapi_generator_ver='5.2.0'
goswagger_ver='v0.23.0'
protoc_ver='3.18.1'
protoc_gen_go_ver='v1.26.0'
protoc_gen_go_grpc='v1.1.0'
richgo_ver='v0.3.10'
mockery_ver='v1.0.0'
mockgen_ver='v1.6.0'
golangcilint_ver='1.33.0'
yamlinc_ver='0.1.10'
node_ver='14.18.2'
dlv_ver='v1.8.1'
gdlv_ver='v1.7.0'
sphinx_ver='4.4.0'
bundler_ver='2.3.8'

# System-dependent variables
case OS
when "macos"
  go_suffix="darwin-amd64"
  goswagger_suffix="darwin_amd64"
  protoc_suffix="osx-x86_64"
  node_suffix="darwin-x64"
  golangcilint_suffix="darwin-amd64"
  chrome_drv_suffix="mac64"
  puts "WARNING: MacOS is not officially supported, the provisions for building on MacOS are made"
  puts "WARNING: for the developers' convenience only."
when "linux"
  go_suffix="linux-amd64"
  goswagger_suffix="linux_amd64"
  protoc_suffix="linux-x86_64"
  node_suffix="linux-x64"
  golangcilint_suffix="linux-amd64"
  chrome_drv_suffix="linux64"
when "FreeBSD"
  goswagger_suffix=""
  puts "WARNING: There are no FreeBSD packages for GOSWAGGER"
  go_suffix="freebsd-amd64"
  # TODO: there are no protoc built packages for FreeBSD (at least as of 3.10.0)
  protoc_suffix=""
  puts "WARNING: There are no protoc packages built for FreeBSD"
  node_suffix="node-v14.18.2.tar.xz"
  golangcilint_suffix="freebsd-amd64"
  chrome_drv_suffix=""
  puts "WARNING: There are no chrome drv packages built for FreeBSD"
else
  puts "ERROR: Unknown/unsupported OS: %s" % UNAME
  fail
end

### Detect Chrome
# CHROME_BIN is required for UI unit tests and system tests. If it is
# not provided by a user, try to locate Chrome binary and set
# environment variable to its location.
if !ENV['CHROME_BIN'] || ENV['CHROME_BIN'].empty?
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

$python_tools_dir = File.join(tools_dir, "python")
directory $python_tools_dir

$pythonpath = File.join($python_tools_dir, "lib")
directory $pythonpath

# Automatically created directories by tools
ruby_tools_gems_dir = File.join(ruby_tools_dir, "gems")
node_bin_dir = File.join(node_dir, "bin")
goroot = File.join(go_tools_dir, "go")
gobin = File.join(goroot, "bin")

# Environment variables
ENV["GEM_HOME"] = ruby_tools_dir
ENV["BUNDLE_PATH"] = ruby_tools_dir
ENV["BUNDLE_BIN"] = ruby_tools_bin_bundle_dir
ENV["GOROOT"] = goroot
ENV["GOPATH"] = gopath
ENV["GOBIN"] = gobin
ENV["PATH"] = "#{node_bin_dir}:#{tools_dir}:#{gobin}:#{ENV["PATH"]}"
ENV["PYTHONPATH"] = $pythonpath

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
        sh *WGET, "https://nodejs.org/dist/v#{node_ver}/node-v#{node_ver}-#{node_suffix}.tar.xz", "-O", "node.tar.xz"
        sh "tar", "-Jxf", "node.tar.xz", "--strip-components=1"
        sh "rm", "node.tar.xz"
    end
    sh NPM, "--version"
end

NPX = File.join(node_bin_dir, "npx")
file NPX => [NPM] do
    sh NPX, "--version"
end

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

# Chrome driver is not currently used, but it can be needed in the UI tests.
# This file task is ready to use after uncomment.
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
file OPENAPI_GENERATOR => tools_dir do
    sh *WGET, "https://repo1.maven.org/maven2/org/openapitools/openapi-generator-cli/#{openapi_generator_ver}/openapi-generator-cli-#{openapi_generator_ver}.jar", "-O", OPENAPI_GENERATOR
end

GO = File.join(gobin, "go")
file GO => [go_tools_dir] do
    Dir.chdir(go_tools_dir) do
        sh *WGET, "https://dl.google.com/go/go#{go_ver}.#{go_suffix}.tar.gz", "-O", "go.tar.gz"
        sh "tar", "-zxf", "go.tar.gz" 
        sh "rm", "go.tar.gz"
    end
    sh GO, "version"
end

GOSWAGGER = File.join(go_tools_dir, "goswagger")
file GOSWAGGER => [go_tools_dir] do
    sh *WGET, "https://github.com/go-swagger/go-swagger/releases/download/#{goswagger_ver}/swagger_#{goswagger_suffix}", "-O", GOSWAGGER
    sh "chmod", "u+x", GOSWAGGER
    sh GOSWAGGER, "version"
end

PROTOC = File.join(go_tools_dir, "protoc")
file PROTOC => [go_tools_dir] do
    Dir.chdir(go_tools_dir) do
        sh *WGET, "https://github.com/protocolbuffers/protobuf/releases/download/v#{protoc_ver}/protoc-#{protoc_ver}-#{protoc_suffix}.zip", "-O", "protoc.zip"
        sh "unzip", "-j", "protoc.zip", "bin/protoc"
        sh "rm", "protoc.zip"
    end
    sh PROTOC, "--version"
end

PROTOC_GEN_GO = File.join(gobin, "protoc-gen-go")
file PROTOC_GEN_GO => [GO] do
    sh GO, "install", "google.golang.org/protobuf/cmd/protoc-gen-go@#{protoc_gen_go_ver}"
    sh PROTOC_GEN_GO, "--version"
end

PROTOC_GEN_GO_GRPC = File.join(gobin, "protoc-gen-go-grpc")
file PROTOC_GEN_GO_GRPC => [GO] do
    sh GO, "install", "google.golang.org/grpc/cmd/protoc-gen-go-grpc@#{protoc_gen_go_grpc}"
    sh PROTOC_GEN_GO_GRPC, "--version"
end

GOLANGCILINT = File.join(go_tools_dir, "golangci-lint")
file GOLANGCILINT => [go_tools_dir] do
    Dir.chdir(go_tools_dir) do
        sh *WGET, "https://github.com/golangci/golangci-lint/releases/download/v#{golangcilint_ver}/golangci-lint-#{golangcilint_ver}-#{golangcilint_suffix}.tar.gz", "-O", "golangci-lint.tar.gz"
        sh "mkdir", "tmp"
        sh "tar", "-zxf", "golangci-lint.tar.gz", "-C", "tmp", "--strip-components=1"
        sh "mv", "tmp/golangci-lint", "."
        sh "rm", "-rf", "tmp"
        sh "rm", "-f", "golangci-lint.tar.gz"
    end
    sh GOLANGCILINT, "--version"
end

RICHGO = "#{gobin}/richgo"
file RICHGO => [GO] do
    sh GO, "install", "github.com/kyoh86/richgo@#{richgo_ver}"
    sh RICHGO, "version"
end

MOCKERY = "#{gobin}/mockery"
file MOCKERY => [GO] do
    sh GO, "get", "github.com/vektra/mockery/.../@#{mockery_ver}"
    sh MOCKERY, "--version"
end

MOCKGEN = "#{gobin}/mockgen"
file MOCKGEN => [GO] do
    sh GO, "install", "github.com/golang/mock/mockgen@#{mockgen_ver}"
    sh MOCKGEN, "--version"
end

DLV = "#{gobin}/dlv"
file DLV => [GO] do
    sh GO, "install", "github.com/go-delve/delve/cmd/dlv@#{dlv_ver}"
    sh DLV, "version"
end

GDLV = "#{gobin}/gdlv"
file GDLV => [GO] do
    sh GO, "install", "github.com/aarzilli/gdlv@#{gdlv_ver}"
    if !File.file?(GDLV)
        fail
    end
end

sphinx_path = File.expand_path("tools/python/bin/sphinx-build")
if ENV["OLD_CI"] == "yes"
    python_location = which("python3")
    python_bin_dir = File.dirname(python_location)
    sphinx_path = File.join(python_bin_dir, "sphinx-build")
end
sphinx_requirements_file = File.expand_path("init_deps/sphinx.txt", __dir__)
SPHINX_BUILD = sphinx_path
file SPHINX_BUILD => [$python_tools_dir, sphinx_requirements_file] do
    if ENV["OLD_CI"] == "yes"
        sh "touch", "-c", SPHINX_BUILD
        next
    end
    pip_install(sphinx_requirements_file)
    sh SPHINX_BUILD, "--version"
end

pytests_path = File.expand_path("tools/python/bin/pytest")
pytests_requirements_file = File.expand_path("init_deps/pytest.txt", __dir__)
PYTEST = pytests_path
file PYTEST => [$python_tools_dir, pytests_requirements_file] do
    pip_install(pytests_requirements_file)
    sh PYTEST, "--version"
end

######################
### Internal tasks ###
######################

# Install Python dependencies from requirements.txt file
def pip_install(requirements_file)
    Rake::FileTask[$python_tools_dir].invoke()
    Rake::FileTask[$pythonpath].invoke()

    ci_opts = []
    if ENV["CI"] == "true"
        ci_opts += ["--no-cache-dir"]
    end
    # Fix for Keyring error with pip. https://github.com/pypa/pip/issues/7883
    ENV["PYTHON_KEYRING_BACKEND"] = "keyring.backends.null.Keyring"
    sh "pip3", "install",
            *ci_opts,
            "--force-reinstall",
            "--upgrade",
            "--no-input",
            "--no-deps",
            # In Python 3.9 ang higher the target option can be used
            "--prefix", $python_tools_dir,
            "--ignore-installed",
            # "--target", $python_tools_dir
            "-r", requirements_file

    # pip install "--target" option doesn't include bin
    # directory for Python < 3.9 version.
    # To workaround this problem, the "--prefix" option
    # is used, but it causes the library path to contain
    # the Python version.
    python_version_out = `python3 --version`
    python_version = (python_version_out.split)[1]
    python_major_minor = python_version.split(".")[0,2].join(".")
    site_packages_dir = File.join($python_tools_dir, "lib", "python" + python_major_minor, "site-packages")
    sh "cp", "-a", site_packages_dir + "/.", $pythonpath
    sh "rm", "-rf", site_packages_dir
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
    check_deps(__FILE__, "wget", "python3", "pip3", "java", "unzip", "entr", "git",
        "createdb", "psql", "dropdb", ENV['CHROME_BIN'], "docker-compose",
        "docker", "gem")
end
