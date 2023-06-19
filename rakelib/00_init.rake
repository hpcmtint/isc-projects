# coding: utf-8

# Initialization
# This file contains the toolkits that
# aren't related to the source code.
# It means that they don't change very often
# and can be cached for later use.

require 'open3'

# Ruby has a built-in solution for handling CLEAN and CLOBBER arrays and
# deleting unnecessary files. But loading the 'rake' module significantly reduces
# the performance. For these reason we implement the clean and clobber tasks
# ourselves.
#
# Clean up the project by deleting scratch files and backup files. Add files to
# the CLEAN FileList to have the clean target handle them.
# Unlike the standard Rake Clean task, this implementation recursively removes
# the directories.
CLEAN = FileList[]
# Clobber all generated and non-source files in a project. The task depends on
# clean, so all the CLEAN files will be deleted as well as files in the CLOBBER
# FileList. The intent of this task is to return a project to its pristine,
# just unpacked state.
CLOBBER = FileList[]

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

# Indicates if the provided task should be considered as a dependency.
# The dependencies are all file tasks and some particular standard tasks that
# define the custom logic to set up and check the dependency. They are
# recognized by the existing :@manuall_install variable.
def is_dependency_task(t)
    t.class == Rake::FileTask || !t.instance_variable_get(:@manuall_install).nil?
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
        if !is_dependency_task(t)
            next
        end

        if t.instance_variable_get(:@manuall_install)
            # Skips the missing top-level manually-installed prerequisites
            # to avoid interrupting preparing operation. If the
            # manually-installed prerequisite is a dependency of any
            # prerequisites and it's missing, then the preparing operation is
            # stopped anyway.
            if which(t.to_s).nil?
                puts "Preparing: #{t.name}... must be manually installed"
                next
            end
        end

        print "Preparing: ", t, "...\n"
        t.invoke()
    end
end

# Searches for the prerequisites from the init file in the provided file and
# checks if they exist.
def check_deps(file)
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

    puts "Self-installed dependencies:"
    prerequisites_tasks.sort_by{ |t| t.name().rpartition("/")[2] }.each do |t|
        if !is_dependency_task(t)
            next
        end

        path = t.to_s
        name = path
        _, _, name = path.rpartition("/")

        print_status(name, path, !t.needed?)
    end

    puts "\nManually-installed dependencies:"

    manual_install_prerequisites_tasks
        .map { |p| [p.name().rpartition("/")[2], p.name(), !p.needed? ] }
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

# Defines a file task with no logic and always has the "not needed" status.
# The file task is rebuilt if the task target is updated (the modification date
# is later than remembered by Rake) or if any prerequisites are updated. The
# task that was changed has the "needed" status in Rake. The tasks created
# using this function are always up-to-date and don't trigger the rebuild of
# the parent task. They always have a "not needed" status and timestamp earlier
# than the parent task.
def create_not_needed_file_task(task_name)
    file task_name

    Rake::Task[task_name].tap do |task|
        def task.timestamp # :nodoc:
            Time.at 0
        end

        def task.needed?
            false
        end
    end

    return task_name
end

# This is a regular task that does nothing. It is dedicated to using as
# prerequirement. Due to it being a regular task it is always recognized as
# "needed" and causes to rebuild of a parent task.
# The file tasks with this prerequisite cannot be a prerequisite for other file
# tasks to avoid a negative impact on the performance.
#
# This task works similarly to a default "phony" task in Ruby but has
# additional validation that prevents you from using it in the middle of the
# dependency chain. It isn't necessary; it changes nothing in how this task
# works, but it verifies if a developer didn't misuse it and provide a
# hard-to-find bug. See discussion about the "phony" task in
# https://gitlab.isc.org/isc-projects/stork/-/merge_requests/535#note_344019.
task :always_rebuild_this_task do |this|
    # Checks if no file task depends on a file task with this prerequisite.
    Rake::Task.tasks().each do |t|
        if !is_dependency_task(t)
            next
        end

        # Iterates over the prerequisities of the file task.
        t.prerequisites.each do |p|
            # Iterates over the nested prerequisities (prerequisities of
            # prerequisities).
            Rake::Task[p].all_prerequisite_tasks.each do |n|
                if n == this
                    fail "#{this} cannot be a prerequsite of file task (#{t})"
                end
            end
        end
    end
end

# Create a new file task that fails if the executable doesn't exist.
# It accepts a path to the executable.
def create_manually_installed_file_task(path)
    file path do
        # This check allows to use manually installed tasks as prerequisities of
        # other manually installed tasks.
        if which(path).nil?
            fail "#{path} must be installed manually on your operating system"
        end
    end

    # Add a property to indicate that it's a manually installed file.
    newTask = Rake::Task[path]
    newTask.instance_variable_set(:@manuall_install, true)
    return newTask.name
end

# Task name should be a path to file or an executable name.
#
# If the path is used, it must be a name of the existing path. If all conditions
# are false, the function leaves the task and task name untouched.
#
# If the task name is a name of the executable (no slashes) then it may not
# have related file task. If all conditions are false, the function creates
# a dump/phony task and leaves the task name untouched.
#
# If any condition is true, the original task is removed. It is replaced with
# a new one that fails if the task target doesn't exist. The task has a property
# that indicates that it requires a manual install. Function returns a new name
# that depends on the which command output.
def require_manual_install_on(task_name, *conditions)
    task = nil
    if Rake::Task.task_defined? task_name
        task = Rake::Task[task_name]
    end

    # The task may not exist for the executables that must be found in PATH.
    # Other files must have assigned file tasks.
    if (!task.nil? && task.class != Rake::FileTask) || (task.nil? && task_name.include?("/"))
        fail "file task required"
    end

    if !conditions.any?
        if task.nil?
            # Create an empty file task to prevent failure due to a non-existing
            # file if the executable isn't prerequisite.
            create_not_needed_file_task(task_name)
        end
        return task_name
    end

    # Remove the self-installed task when it is unsupported.
    if !task.nil?
        task.clear()
        Rake.application.instance_variable_get('@tasks').delete(task_name)
    end

    # Search in PATH for executable.
    program = File.basename task_name
    system_path = which(program)
    if !system_path.nil?
        program = system_path
    end

    # Create a new task that fails if the executable doesn't exist.
    return create_manually_installed_file_task(program)
end

# Fetches the file from the network. You should add the WGET to the
# prerequisites of the task that uses this function.
# The file is saved in the target location.
def fetch_file(url, target)
    # extract wget version
    stdout, _, status = Open3.capture3(WGET, "--version")
    wget = [WGET]

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
        # Suppress verbose output on the CI.
        wget.append "--no-verbose"
    end

    wget.append url
    wget.append "-O", target

    sh *wget
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
any_system = true

### Define package versions
go_ver='1.19.7'
goswagger_ver='v0.30.4'
protoc_ver='3.20.3'
protoc_gen_go_ver='v1.30.0'
protoc_gen_go_grpc_ver='v1.3.0'
richgo_ver='v0.3.12'
mockery_ver='v2.20.2'
mockgen_ver='v1.6.0'
golangcilint_ver='1.51.2'
dlv_ver='v1.20.1'
gdlv_ver='v1.9.0'
openapi_generator_ver='6.4.0'
# NodeJS major update blocked due to out-of-date CI operating system.
node_ver='14.21.3'
npm_ver='9.6.2'
storybook_ver='6.5.16'
yamlinc_ver='0.1.10'
bundler_ver='2.3.8'
shellcheck_ver='0.9.0'

# System-dependent variables
case OS
when "macos"
  go_suffix="darwin-amd64"
  protoc_suffix="osx-x86_64"
  node_suffix="darwin-x64"
  golangcilint_suffix="darwin-amd64"
  chrome_drv_suffix="mac64"
  shellcheck_suffix="darwin.x86_64"
  puts "WARNING: MacOS is not officially supported, the provisions for building on MacOS are made"
  puts "WARNING: for the developers' convenience only."
when "linux"
  go_suffix="linux-amd64"
  protoc_suffix="linux-x86_64"
  node_suffix="linux-x64"
  golangcilint_suffix="linux-amd64"
  chrome_drv_suffix="linux64"
  shellcheck_suffix="linux.x86_64"
when "FreeBSD"
  go_suffix="freebsd-amd64"
  golangcilint_suffix="freebsd-amd64"
when "OpenBSD"
else
  puts "ERROR: Unknown/unsupported OS: %s" % UNAME
  fail
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

if libc_musl_system || openbsd_system
    gobin = ENV["GOBIN"]
    goroot = ENV["GOROOT"]
    if gobin.nil?
        gobin = which("go")
        if !gobin.nil?
            gobin = File.dirname gobin
        else
            gobin = ""
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

### Detect Chrome
# CHROME_BIN is required for UI unit tests and system tests. If it is
# not provided by a user, try to locate Chrome binary and set
# environment variable to its location.
def detect_chrome_binary()
    if !ENV['CHROME_BIN'].nil? && !ENV['CHROME_BIN'].empty?
        location = which(ENV['CHROME_BIN'])
        if !location.nil?
            return location
        end
    end

    location = which("chromium")
    if !location.nil?
        return location
    end

    location = which("chrome")
    if !location.nil?
        return location
    end

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
            return loc
        end
    end

    return nil
end

CHROME = create_manually_installed_file_task(detect_chrome_binary() || "chrome")
ENV["CHROME_BIN"] = CHROME

# System tools
WGET = require_manual_install_on("wget", any_system)
PYTHON3_SYSTEM = require_manual_install_on("python3", any_system)
JAVA = require_manual_install_on("java", any_system)
UNZIP = require_manual_install_on("unzip", any_system)
ENTR = require_manual_install_on("entr", any_system)
GIT = require_manual_install_on("git", any_system)
CREATEDB = require_manual_install_on("createdb", any_system)
PSQL = require_manual_install_on("psql", any_system)
DROPDB = require_manual_install_on("dropdb", any_system)
DROPUSER = require_manual_install_on("dropuser", any_system)
DOCKER = require_manual_install_on("docker", any_system)
OPENSSL = require_manual_install_on("openssl", any_system)
GEM = require_manual_install_on("gem", any_system)
MAKE = require_manual_install_on("make", any_system)
GCC = require_manual_install_on("gcc", any_system)
TAR = require_manual_install_on("tar", any_system)
SED = require_manual_install_on("sed", any_system)
PERL = require_manual_install_on("perl", any_system)
FOLD = require_manual_install_on("fold", any_system)
SSH = require_manual_install_on("ssh", any_system)
SCP = require_manual_install_on("scp", any_system)
CLOUDSMITH = require_manual_install_on("cloudsmith", any_system)
ETAGS_CTAGS = require_manual_install_on("etags.ctags", any_system)
CLANGPLUSPLUS = require_manual_install_on("clang++", openbsd_system)

# Docker compose requirement task
task :docker_compose => [DOCKER] do
    # The docker compose (or docker-compose) must be manually installed. If it
    # is installed, the task body is never called.
    fail "docker compose plugin or docker-compose standalone is not installed"
end

# The constant to use as prerequisites.
DOCKER_COMPOSE = Rake::Task[:docker_compose]

Rake::Task[:docker_compose].tap do |task|
    # The non-file tasks with the manuall_install variable are considered as
    # dependencies. Set to true to mark it must be manually installed.
    task.instance_variable_set(:@manuall_install, true)

    # Check if the docker compose plugin is installed.
    def is_docker_compose_v2_supported()
        begin
            _, _, status = Open3.capture3 DOCKER, "compose"
            return status == 0
        rescue
            # Missing docker command in system.
            return false
        end
    end

    # Check if the standalone docker-compose is installed.
    def is_docker_compose_v1_supported()
        return !which("docker-compose").nil?
    end

    # Check if the task should be called. It is internally called by Rake.
    # Return false if the docker compose or docker-compose is ready to use.
    def task.needed?
        # Fail the task if the docker compose or docker-compose is missing.
        return !is_docker_compose_v2_supported() && !is_docker_compose_v1_supported()
    end

    # Return the string representation of the task.
    def task.name
        if is_docker_compose_v2_supported() || !is_docker_compose_v1_supported()
            return "#{DOCKER} compose"
        else
            return which("docker-compose")
        end
    end

    # Return the task name. It is used for compatibility with Rake. The
    # identifier of the task and the to_s output must be the same.
    def task.to_s
        return :docker_compose.to_s
    end

    # Handle the splat operator call (*task_name). The splat operator should
    # be used to call the task-related command.
    # E.g.: sh *DOCKER_COMPOSE, --foo, --bar
    def task.to_a
        if is_docker_compose_v2_supported() || !is_docker_compose_v1_supported()
            return [DOCKER, "compose"]
        else
            return [which("docker-compose")]
        end
    end
end

# Toolkits
BUNDLE = File.join(ruby_tools_bin_dir, "bundle")
file BUNDLE => [GEM, ruby_tools_dir, ruby_tools_bin_dir] do
    sh GEM, "install",
            "--minimal-deps",
            "--no-document",
            "--no-user-install",
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

fpm_gemfile = File.expand_path("init_deps/fpm/Gemfile", __dir__)
FPM = File.join(ruby_tools_bin_bundle_dir, "fpm")
file FPM => [BUNDLE, ruby_tools_dir, ruby_tools_bin_bundle_dir, fpm_gemfile] do
    sh BUNDLE, "install",
        "--gemfile", fpm_gemfile,
        "--path", ruby_tools_dir,
        "--binstubs", ruby_tools_bin_bundle_dir
    sh FPM, "--version"
end

danger_gemfile = File.expand_path("init_deps/danger/Gemfile", __dir__)
DANGER = File.join(ruby_tools_bin_bundle_dir, "danger")
file DANGER => [ruby_tools_bin_bundle_dir, ruby_tools_dir, danger_gemfile, BUNDLE] do
    sh BUNDLE, "install",
        "--gemfile", danger_gemfile,
        "--path", ruby_tools_dir,
        "--binstubs", ruby_tools_bin_bundle_dir
    sh "touch", "-c", DANGER
    sh DANGER, "--version"
end

node = File.join(node_bin_dir, "node")
file node => [TAR, WGET, node_dir] do
    Dir.chdir(node_dir) do
        FileUtils.rm_rf(FileList["*"])
        fetch_file "https://nodejs.org/dist/v#{node_ver}/node-v#{node_ver}-#{node_suffix}.tar.xz", "node.tar.xz"
        sh TAR, "-Jxf", "node.tar.xz", "--strip-components=1"
        sh "rm", "node.tar.xz"
    end
    sh "touch", "-c", node
    sh node, "--version"
end
node = require_manual_install_on(node, libc_musl_system, freebsd_system, openbsd_system)
add_version_guard(node, node_ver)

npm = File.join(node_bin_dir, "npm")
file npm => [node] do
    ci_opts = []
    if ENV["CI"] == "true"
        ci_opts += ["--no-audit", "--no-progress"]
    end

    # NPM is initially installed with NodeJS.
    sh npm, "install",
            "-g",
            *ci_opts,
            "npm@#{npm_ver}"
    sh "touch", "-c", npm
    sh npm, "--version"
end
NPM = require_manual_install_on(npm, libc_musl_system, freebsd_system, openbsd_system)
add_version_guard(NPM, npm_ver)

npx = File.join(node_bin_dir, "npx")
file npx => [NPM] do
    sh npx, "--version"
    sh "touch", "-c", npx
end
NPX = require_manual_install_on(npx, libc_musl_system, freebsd_system, openbsd_system)

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
# file CHROME_DRV => [WGET, UNZIP, tools_dir] do
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
#         fetch_file "https://chromedriver.storage.googleapis.com/#{chrome_drv_version}/chromedriver_#{chrome_drv_suffix}.zip", "chromedriver.zip"
#         sh UNZIP, "-o", "chromedriver.zip"
#         sh "rm", "chromedriver.zip"
#     end

#     sh CHROME_DRV, "--version"
#     sh "chromedriver", "--version"  # From PATH
# end

OPENAPI_GENERATOR = File.join(tools_dir, "openapi-generator-cli.jar")
file OPENAPI_GENERATOR => [WGET, tools_dir] do
    fetch_file "https://repo1.maven.org/maven2/org/openapitools/openapi-generator-cli/#{openapi_generator_ver}/openapi-generator-cli-#{openapi_generator_ver}.jar", OPENAPI_GENERATOR
    sh "touch", "-c", OPENAPI_GENERATOR
end
add_version_guard(OPENAPI_GENERATOR, openapi_generator_ver)

go = File.join(gobin, "go")
file go => [WGET, go_tools_dir] do
    Dir.chdir(go_tools_dir) do
        FileUtils.rm_rf("go")
        fetch_file "https://dl.google.com/go/go#{go_ver}.#{go_suffix}.tar.gz", "go.tar.gz"
        sh "tar", "-zxf", "go.tar.gz"
        sh "rm", "go.tar.gz"
    end
    sh "touch", "-c", go
    sh go, "version"
end
GO = require_manual_install_on(go, libc_musl_system, openbsd_system)
add_version_guard(GO, go_ver)

GOSWAGGER = File.join(go_tools_dir, "goswagger")
file GOSWAGGER => [WGET, GO, TAR, go_tools_dir] do
    if OS != 'FreeBSD' && OS != "OpenBSD"
        goswagger_suffix = "linux_amd64"
        if OS == 'macos'
            # GoSwagger fails to build on macOS due to https://gitlab.isc.org/isc-projects/stork/-/issues/848.
            goswagger_suffix="darwin_amd64"
        end
        fetch_file "https://github.com/go-swagger/go-swagger/releases/download/#{goswagger_ver}/swagger_#{goswagger_suffix}", GOSWAGGER
        sh "chmod", "u+x", GOSWAGGER
    else
        # GoSwagger lacks the packages for BSD-like systems then it must be
        # built from sources.
        goswagger_archive = "#{GOSWAGGER}.tar.gz"
        goswagger_dir = "#{GOSWAGGER}-sources"
        sh "mkdir", goswagger_dir
        fetch_file "https://github.com/go-swagger/go-swagger/archive/refs/tags/#{goswagger_ver}.tar.gz", goswagger_archive
        sh TAR, "-zxf", goswagger_archive, "-C", goswagger_dir
        # We cannot use --strip-components because OpenBSD tar doesn't support it.
        goswagger_dir = File.join(goswagger_dir, "go-swagger-#{goswagger_ver[1..-1]}") # Trim 'v' letter
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

protoc = File.join(protoc_dir, "protoc")
file protoc => [WGET, UNZIP, go_tools_dir] do
    Dir.chdir(go_tools_dir) do
        fetch_file "https://github.com/protocolbuffers/protobuf/releases/download/v#{protoc_ver}/protoc-#{protoc_ver}-#{protoc_suffix}.zip", "protoc.zip"
        sh UNZIP, "-o", "-j", "protoc.zip", "bin/protoc"
        sh "rm", "protoc.zip"
    end
    sh protoc, "--version"
    sh "touch", "-c", protoc
end
PROTOC = require_manual_install_on(protoc, libc_musl_system, freebsd_system, openbsd_system)
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

golangcilint = File.join(go_tools_dir, "golangci-lint")
file golangcilint => [WGET, GO, TAR, go_tools_dir] do
    Dir.chdir(go_tools_dir) do
        fetch_file "https://github.com/golangci/golangci-lint/releases/download/v#{golangcilint_ver}/golangci-lint-#{golangcilint_ver}-#{golangcilint_suffix}.tar.gz", "golangci-lint.tar.gz"
        sh "mkdir", "tmp"
        sh TAR, "-zxf", "golangci-lint.tar.gz", "-C", "tmp", "--strip-components=1"
        sh "mv", "tmp/golangci-lint", "."
        sh "rm", "-rf", "tmp"
        sh "rm", "-f", "golangci-lint.tar.gz"
    end
    sh "touch", "-c", golangcilint
    sh golangcilint, "--version"
end
GOLANGCILINT = require_manual_install_on(golangcilint, openbsd_system)
add_version_guard(GOLANGCILINT, golangcilint_ver)

shellcheck = File.join(tools_dir, "shellcheck")
file shellcheck => [WGET, TAR, tools_dir] do
    Dir.chdir(tools_dir) do
        # Download the shellcheck binary.
        fetch_file "https://github.com/koalaman/shellcheck/releases/download/v#{shellcheck_ver}/shellcheck-v#{shellcheck_ver}.#{shellcheck_suffix}.tar.xz", "shellcheck.tar.xz"
        sh "mkdir", "-p", "tmp"
        sh TAR, "-xf", "shellcheck.tar.xz", "-C", "tmp", "--strip-components=1"
        sh "mv", "tmp/shellcheck", "."
        sh "rm", "-rf", "tmp"
        sh "rm", "-f", "shellcheck.tar.xz"
    end
    sh "touch", "-c", shellcheck
    sh shellcheck, "--version"
end
SHELLCHECK = require_manual_install_on(shellcheck, freebsd_system, openbsd_system)
add_version_guard(SHELLCHECK, shellcheck_ver)

RICHGO = "#{gobin}/richgo"
file RICHGO => [GO] do
    sh GO, "install", "github.com/kyoh86/richgo@#{richgo_ver}"
    sh RICHGO, "version"
end
add_version_guard(RICHGO, richgo_ver)

MOCKERY = File.join(gobin, "mockery")
file MOCKERY => [GO] do
    sh GO, "install", "github.com/vektra/mockery/v2@#{mockery_ver}"
    sh MOCKERY, "--version"
end
add_version_guard(MOCKERY, mockery_ver)

MOCKGEN = File.join(gobin, "mockgen")
file MOCKGEN => [GO] do
    sh GO, "install", "github.com/golang/mock/mockgen@#{mockgen_ver}"
    sh MOCKGEN, "--version"
end
add_version_guard(MOCKGEN, mockgen_ver)

DLV = File.join(gobin, "dlv")
file DLV => [GO] do
    sh GO, "install", "github.com/go-delve/delve/cmd/dlv@#{dlv_ver}"
    sh DLV, "version"
end
add_version_guard(DLV, dlv_ver)

GDLV = File.join(gobin, "gdlv")
file GDLV => [GO] do
    sh GO, "install", "github.com/aarzilli/gdlv@#{gdlv_ver}"
    if !File.file?(GDLV)
        fail
    end
end
add_version_guard(GDLV, gdlv_ver)

GOVULNCHECK = File.join(gobin, "govulncheck")
file GOVULNCHECK => [GO, :always_rebuild_this_task] do
    # Govulncheck is still in the development phase. It doesn't use stable
    # tags. Available versions have a short lifetime. We use the latest release
    # and mark it as always out-of-date for Rake. The Go will check if the
    # newer version is available every run. It shouldn't be problematic as long
    # as it will be used only in a task to check the vulnerabilities.
    sh GO, "install", "golang.org/x/vuln/cmd/govulncheck@latest"
    if !File.file?(GOVULNCHECK)
        fail
    end
end

PYTHON = File.join(python_tools_dir, "bin", "python")
file PYTHON => [PYTHON3_SYSTEM] do
    sh PYTHON3_SYSTEM, "-m", "venv", python_tools_dir
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

PIP_COMPILE = File.expand_path("tools/python/bin/pip-compile")
file PIP_COMPILE => [PIP] do
    sh PIP, "install", "pip-tools"
    sh "touch", "-c", PIP_COMPILE
    sh PIP_COMPILE, "--version"
end

PYLINT = File.expand_path("tools/python/bin/pylint")
python_linters_requirements_file = File.expand_path("init_deps/pylinters.txt", __dir__)
file PYLINT => [PIP, python_linters_requirements_file] do
    sh PIP, "install", "-r", python_linters_requirements_file
    sh "touch", "-c", PYLINT
    sh PYLINT, "--version"
end

FLAKE8 = File.expand_path("tools/python/bin/flake8")
file FLAKE8 => [PYLINT] do
    sh "touch", "-c", FLAKE8
    sh FLAKE8, "--version"
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
    check_deps(__FILE__)
end
