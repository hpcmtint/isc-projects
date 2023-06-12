# Hooks
# The file contains tasks to write and build the Stork hook libraries
# (GO plugins)

#############
### Files ###
#############

_main_module = "isc.org/stork"
_main_module_with_version = "#{_main_module}@v0.0.0"
_main_remote_repository_url = "gitlab.isc.org/isc-projects/stork/backend"

default_hook_directory_rel = "hooks"
DEFAULT_HOOK_DIRECTORY = File.expand_path default_hook_directory_rel

CLEAN.append *FileList[File.join(DEFAULT_HOOK_DIRECTORY, "*.so")]


#################
### Functions ###
#################

def forEachHook(f)
    hook_directory = ENV["HOOK_DIR"] || DEFAULT_HOOK_DIRECTORY

    Dir.foreach(hook_directory) do |dir_name|
        path = File.join(hook_directory, dir_name)
        next if dir_name == '.' or dir_name == '..' or !File.directory? path

        Dir.chdir(path) do
            f.call(dir_name)
        end
    end
end

#############
### Tasks ###
#############

namespace :hook do
    desc "Init new hook directory
        MODULE - the name  of the hook module used in the go.mod file and as the hook directory name - required
        HOOK_DIR - the directory containing the hooks - optional, default: #{default_hook_directory_rel}"
    task :init => [GO, GIT] do
        module_name = ENV["MODULE"]
        if module_name.nil?
            fail "You must provide the MODULE variable with the module name"
        end

        hook_directory = ENV["HOOK_DIR"] || DEFAULT_HOOK_DIRECTORY
        
        module_directory_name = module_name.gsub(/[^\w\.-]/, '_')

        destination = File.expand_path(File.join(hook_directory, module_directory_name))

        require 'pathname'
        main_module = "isc.org/stork@v0.0.0"
        main_module_directory_abs = Pathname.new('backend').realdirpath
        module_directory_abs = Pathname.new(destination)
        module_directory_rel = main_module_directory_abs.relative_path_from module_directory_abs

        sh "mkdir", "-p", destination

        Dir.chdir(destination) do
            sh GIT, "init"
            sh GO, "mod", "init", module_name
            sh GO, "mod", "edit", "-require", main_module
            sh GO, "mod", "edit", "-replace", "#{main_module}=#{module_directory_rel}"
            sh "touch", "go.sum"
        end
        
        sh "cp", *FileList["backend/hooksutil/boilerplate/*"], destination
    end

    desc "Build all hooks. Remap hooks to use the current codebase.
        DEBUG - build hooks in debug mode, the envvar is passed through to the hook Rakefile - default: false
        HOOK_DIR - the hook (plugin) directory - optional, default: #{default_hook_directory_rel}"
    task :build => [GO, "hook:remap_core:local"] do
        require 'tmpdir'

        hook_directory = ENV["HOOK_DIR"] || DEFAULT_HOOK_DIRECTORY

        # Removes old hooks
        puts "Removing old compiled hooks..."
        sh "rm", "-f", *FileList[File.join(hook_directory, "*.so")]

        mod_files = ["go.mod", "go.sum"]

        forEachHook(lambda { |dir_name|
            # Make a backup of the original mod files
            Dir.mktmpdir do |temp|
                sh "cp", *mod_files, temp

                puts "Building #{dir_name}..."
                sh "rake", "build"
                sh "cp", *FileList["build/*.so"], hook_directory

                # Back the changes in Go mod files.
                puts "Reverting remap operation..."
                sh "cp", *mod_files.collect { |f| File.join(temp, f) }, "."
            end
        })
    end

    desc "Rename the hook files to the conventional names
        HOOK_DIR - the hook (plugin) directory - optional, default: #{default_hook_directory_rel}"
    task :rename => [GIT] do
        hook_directory = ENV["HOOK_DIR"] || DEFAULT_HOOK_DIRECTORY
        # The plugin filenames after remap lack the version.
        # We need to append it.
        commit, _ = Open3.capture2 GIT, "rev-parse", "--short", "HEAD"
        commit = commit.strip()

        Dir[File.join(hook_directory, "*.so")].each do |path|
            new_path = File.join(
                File.dirname(path),
                "#{File.basename(path, ".so")}#{STORK_VERSION}-#{commit}.so"
            )
            sh "mv", path, new_path
        end
    end

    desc "Remap the dependency path to the Stork core. It specifies the source
        of the core dependency - remote repository or local directory. The
        remote repository may be fetched by tag or commit hash.
        HOOK_DIR - the hook (plugin) directory - optional, default: #{default_hook_directory_rel}
        COMMIT - use the given commit from the remote repository, if specified but empty use the current hash - optional
        TAG - use the given tag from the remote repository, if specified but empty use the current version as tag - optional
        If no COMMIT or TAG are specified then it remaps to use the local project."
    task :remap_core do
        if !ENV["COMMIT"].nil?
            puts "Remap to use a specific commit"
            Rake::Task["hook:remap_core:commit"].invoke()
        elsif !ENV["TAG"].nil?
            puts "Remap to use a specific tag"
            Rake::Task["hook:remap_core:tag"].invoke()
        else
            puts "Remap to use the local directory"
            Rake::Task["hook:remap_core:local"].invoke()
        end
    end

    namespace :remap_core do
        desc "Remap the dependency path to the Stork core. It specifies the source
            of the core dependency as remote repository referenced by commit.
            HOOK_DIR - the hook (plugin) directory - optional, default: #{default_hook_directory_rel}
            COMMIT - use the given commit from the remote repository, if specified but empty use the current hash - optional"
        task :commit => [GO, GIT] do
            commit = ENV["COMMIT"]
            if commit.nil? || commit == ""
                commit, _ = Open3.capture2 GIT, "rev-parse", "HEAD"
            end

            target = "#{_main_remote_repository_url}@#{commit}"

            forEachHook(lambda { |dir_name|
                sh GO, "mod", "edit", "-replace", "#{_main_module}=#{target}"
                sh GO, "mod", "tidy"
            })
        end

        desc "Remap the dependency path to the Stork core. It specifies the source
            of the core dependency as remote repository referenced by tag.
            HOOK_DIR - the hook (plugin) directory - optional, default: #{default_hook_directory_rel}
            TAG - use the given tag from the remote repository, if specified but empty use the current version as tag - optional"
        task :tag => [GO] do
            tag = ENV["TAG"]
            if tag.nil? || tag == ""
                tag = STORK_VERSION
            end

            if !tag.start_with? "v"
                tag = "v" + tag
            end

            target = "#{_main_remote_repository_url}@#{tag}"

            forEachHook(lambda { |dir_name|
                sh GO, "mod", "edit", "-replace", "#{_main_module}=#{target}"
                sh GO, "mod", "tidy"
            })
        end

        desc "Remap the dependency path to the Stork core. It specifies the source
            of the core dependency as local repository."
        task :local => [GO] do
            main_module_directory_abs = File.expand_path "backend"

            forEachHook(lambda { |dir_name|
                require 'pathname'
                main_directory_abs_obj = Pathname.new(main_module_directory_abs)
                module_directory_abs_obj = Pathname.new(".").realdirpath
                module_directory_rel_obj = main_directory_abs_obj.relative_path_from module_directory_abs_obj

                target = module_directory_rel_obj.to_s

                sh GO, "mod", "edit", "-replace", "#{_main_module}=#{target}"
                sh GO, "mod", "tidy"
            })
        end
    end

    desc "Install hooks dependencies"
    task :prepare => [GO] do
        forEachHook(lambda { |dir_name|
            sh GO, "mod", "download"
        })
    end

    desc "List dependencies of a given callout specification package
        KIND - callout kind - required, choice: agent or server
        CALLOUT - callout specification (interface) package name - required"
    task :list_callout_deps => [GO] do
        kind = ENV["KIND"]
        if kind != "server" && kind != "agent"
            fail "You need to provide the callout kind in KIND variable: agent or server"
        end

        callout = ENV["CALLOUT"]
        if callout.nil?
            fail "You need to provide the callout package name in CALLOUT variable."
        end

        package_rel = "hooks/#{kind}/#{callout}"
        ENV["REL"] = package_rel
        Rake::Task["utils:list_package_deps"].invoke
    end

    desc "Fetch official hook sources
        HOOK_DIR - the hook (plugin) directory - optional, default: #{default_hook_directory_rel}
        HOOK - name of the hook (e.g: ldap, example) - optional, if not provided
               all hooks are fetched.
    "
    task :fetch_official => [GIT] do
        hook_directory = ENV["HOOK_DIR"] || DEFAULT_HOOK_DIRECTORY
        requested_hook = nil
        hook_prefix = 'stork-hook-'
        if !ENV["HOOK"].nil?
            requested_hook = "#{hook_prefix}#{ENV["HOOK"]}"
        end

        official_hook_repositories = [
            'git@gitlab.isc.org:isc-projects/stork-hook-ldap.git',
            'git@gitlab.isc.org:isc-projects/stork-hook-example.git'
        ]

        official_hook_repositories.each do |url|
            directory_start_idx = url.rindex('/') + 1
            directory_end_idx = -('.git'.length + 1)
            directory_name = url[directory_start_idx..directory_end_idx]
            hook_name = directory_name[hook_prefix.length..-1]

            Dir.chdir hook_directory do
                if !requested_hook.nil? && requested_hook != directory_name
                    puts "Hook '#{hook_name}' is not selected. Skip."
                    next
                end

                directory_path = File.expand_path directory_name
                if File.exists? directory_path
                    puts "Hook '#{hook_name}' is already fetched. Skip."
                    next
                end

                puts "Clone '#{hook_name}' hook..."
                sh GIT, "clone", url
            end
        end
    end
end

namespace :run do
    desc "Run Stork Server with hooks
        HOOK_DIR - the hook (plugin) directory - optional, default: #{default_hook_directory_rel}"
    task :server_hooks => ["hook:build"] do
        hook_directory = ENV["HOOK_DIR"] || ENV["STORK_SERVER_HOOK_DIRECTORY"] || DEFAULT_HOOK_DIRECTORY
        ENV["STORK_SERVER_HOOK_DIRECTORY"] = hook_directory
        Rake::Task["run:server"].invoke()
    end
end
