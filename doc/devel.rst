.. _devel:

*****************
Developer's Guide
*****************

.. note::

   ISC acknowledges that users and developers have different needs, so
   the user and developer documents should eventually be
   separated. However, since the project is still in its early stages,
   this section is kept in the Stork ARM for convenience.

Rakefile
========

Rakefile is a script for performing many development tasks, like
building source code, running linters and unit tests, and running
Stork services directly or in Docker containers.

There are several other Rake targets. For a complete list of available
tasks, use ``rake -T``. Also see the Stork `wiki
<https://gitlab.isc.org/isc-projects/stork/-/wikis/Processes/development-Environment#building-testing-and-running-stork>`_
for detailed instructions.

Generating Documentation
========================

To generate documentation, simply type ``rake doc``.
`Sphinx <https://www.sphinx-doc.org>`_ and `rtd-theme
<https://github.com/readthedocs/sphinx_rtd_theme>`_ must be installed. The
generated documentation will be available in the ``doc/singlehtml``
directory.

Setting Up the Development Environment
======================================

The following steps install Stork and its dependencies natively,
i.e., on the host machine, rather than using Docker images.

First, PostgreSQL must be installed. This is OS-specific, so please
follow the instructions from the :ref:`installation` chapter.

Once the database environment is set up, the next step is to build all
the tools. The first command below downloads some missing dependencies
and installs them in a local directory. This is done only once
and is not needed for future rebuilds, although it is safe to rerun
the command.

.. code-block:: console

    $ rake build_backend
    $ rake build_ui

The environment should be ready to run. Open three consoles and run
the following three commands, one in each console:

.. code-block:: console

    $ rake run_server

.. code-block:: console

    $ rake serve_ui

.. code-block:: console

    $ rake run_agent

Once all three processes are running, connect to http://localhost:8080
via a web browser. See :ref:`usage` for information on initial password creation
or addition of new machines to the server.

The ``run_agent`` runs the agent directly on the current operating
system, natively; the exposed port of the agent is 8888.

There are other Rake tasks for running preconfigured agents in Docker
containers. They are exposed to the host on specific ports.

When these agents are added as machines in the Stork server UI,
both a localhost address and a port specific to a given container must
be specified. The list of containers can be found in the
:ref:`docker_containers_for_development` section.

Installing Git Hooks
--------------------

There is a simple git hook that inserts the issue number in the commit
message automatically; to use it, go to the ``utils`` directory and
run the ``git-hooks-install`` script. It copies the necessary file
to the ``.git/hooks`` directory.

Agent API
=========

The connection between ``stork-server`` and the agents is established using
gRPC over http/2. The agent API definition is kept in the
``backend/api/agent.proto`` file. For debugging purposes, it is
possible to connect to the agent using the `grpcurl
<https://github.com/fullstorydev/grpcurl>`_ tool. For example, a list
of currently provided gRPC calls may be retrieved with this command:

.. code:: console

    $ grpcurl -plaintext -proto backend/api/agent.proto localhost:8888 describe
    agentapi.Agent is a service:
    service Agent {
      rpc detectServices ( .agentapi.DetectServicesReq ) returns ( .agentapi.DetectServicesRsp );
      rpc getState ( .agentapi.GetStateReq ) returns ( .agentapi.GetStateRsp );
      rpc restartKea ( .agentapi.RestartKeaReq ) returns ( .agentapi.RestartKeaRsp );
    }

Specific gRPC calls can also be made. For example, to get the machine
state, use the following command:

.. code:: console

    $ grpcurl -plaintext -proto backend/api/agent.proto localhost:8888 agentapi.Agent.getState
    {
      "agentVersion": "0.1.0",
      "hostname": "copernicus",
      "cpus": "8",
      "cpusLoad": "1.68 1.46 1.28",
      "memory": "16",
      "usedMemory": "59",
      "uptime": "2",
      "os": "darwin",
      "platform": "darwin",
      "platformFamily": "Standalone Workstation",
      "platformVersion": "10.14.6",
      "kernelVersion": "18.7.0",
      "kernelArch": "x86_64",
      "hostID": "c41337a1-0ec3-3896-a954-a1f85e849d53"
    }

RESTful API
===========

The primary user of the RESTful API is the Stork UI in a web browser. The
definition of the RESTful API is located in the ``api`` folder and is
described in Swagger 2.0 format.

The description in Swagger is split into multiple files. Two files
comprise a tag group:

* \*-paths.yaml - defines URLs
* \*-defs.yaml - contains entity definitions

All these files are combined by the ``yamlinc`` tool into a single
Swagger file, ``swagger.yaml``, which then generates code
for:

* the UI fronted by swagger-codegen
* the backend in Go lang by go-swagger

All these steps are accomplished by Rakefile.

Backend Unit Tests
==================

There are unit tests for the Stork agent and server backends, written in Go.
They can be run using Rake:

.. code:: console

          $ rake unittest_backend

This requires preparing a database in PostgreSQL. One way to avoid
doing this manually is by using a Docker container with PostgreSQL,
which is automatically created when running the following Rake task:

.. code:: console

          $ rake unittest_backend_db

This task spawns a container with PostgreSQL in the background, which
then runs unit tests. When the tests are completed, the database is
shut down and removed.

Unit Tests Database
-------------------

When a Docker container with a database is not used for unit tests, the
PostgreSQL server must be started and the following role must be
created:

.. code-block:: psql

    postgres=# CREATE USER storktest WITH PASSWORD 'storktest';
    CREATE ROLE
    postgres=# ALTER ROLE storktest SUPERUSER;
    ALTER ROLE

To point unit tests to a specific Stork database, set the ``POSTGRES_ADDR``
environment variable, e.g.:

.. code:: console

          $ rake unittest_backend POSTGRES_ADDR=host:port

By default it points to ``localhost:5432``.

Similarly, if the database setup requires a password other than the default
``storktest``,  the ``STORK_DATABASE_PASSWORD`` variable can be used by issuing
the following command:

.. code:: console

          $ rake unittest_backend STORK_DATABASE_PASSWORD=secret123

Note that there is no need to create the ``storktest`` database itself; it is created
and destroyed by the Rakefile task.

Unit Tests Coverage
-------------------

A coverage report is presented once the tests have executed. If
coverage of any module is below a threshold of 35%, an error is
raised.

Benchmarks
----------

Benchmarks are part of backend unit tests. They are implemented using the
golang "testing" library and they test performance-sensitive parts of the
backend. Unlike unit tests, the benchmarks do not return pass/fail status.
They measure average execution time of functions and print the results to
the console.

In order to run unit tests with benchmarks, the ``benchmark`` environment
variable must be specified as follows:

.. code:: console

          $ rake unittest_backend benchmark=true

This command runs all unit tests and all benchmarks. Running benchmarks
without unit tests is possible using the combination of the ``benchmark`` and
``test`` environment variables:

.. code:: console

          $ rake unittest_backend benchmark=true test=Bench

Benchmarks are useful to test the performance of complex functions and find
bottlenecks. When working on improving the performance of a function, examining a
benchmark result before and after the changes is a good practice to ensure
that the goals of the changes are achieved.

Similarly, adding new logic to a function often causes performance
degradation, and careful examination of the benchmark result drop for that
function may drive improved efficiency of the new code.

Short Testing Mode
------------------

It is possible to filter out long-running unit tests, by setting the ``short``
variable to ``true`` on the command line:

.. code:: console

          $ rake unittest_backend short=true


Web UI Unit Tests
=================

Stork offers web UI tests, to take advantage of the unit tests generated automatically
by Angular. The simplest way to run these tests is by using Rake tasks:

.. code:: console

   rake build_ui
   rake ng_test

The tests require the Chromium (on Linux) or Chrome (on Mac) browser. The ``rake ng_test``
task attempts to locate the browser binary and launch it automatically. If the
browser binary is not found in the default location, the Rake task returns an
error. It is possible to set the location manually by setting the ``CHROME_BIN``
environment variable; for example:

.. code:: console

   export CHROME_BIN=/usr/local/bin/chromium-browser
   rake ng_test

By default, the tests launch the browser in headless mode, in which test results
and any possible errors are printed in the console. However, in some situations it
is useful to run the browser in non-headless mode because it provides debugging features
in Chrome's graphical interface. It also allows for selectively running the tests.
Run the tests in non-headless mode using the ``debug`` variable appended to the ``rake``
command:

.. code:: console

   rake ng_test debug=true

That command causes a new browser window to open; the tests run there automatically.

The tests are run in random order by default, which can make it difficult
to chase individual errors. To make debugging easier by always running the tests
in the same order, click "Debug" in the new Chrome window, then click
"Options" and unset the "run tests in random order" button. A specific test can
be run by clicking on its name.

.. code:: console

    test=src/app/ha-status-panel/ha-status-panel.component.spec.ts rake ng_test

By default, all tests are executed. To run only a specific test file,
set the "test" environment variable to a relative path to any ``.spec.ts``
file (relative from the project directory).

When adding a new component or service with ``ng generate component|service ...``, the Angular framework
adds a ``.spec.ts`` file with boilerplate code. In most cases, the first step in
running those tests is to add the necessary Stork imports. If in doubt, refer to the commits on
https://gitlab.isc.org/isc-projects/stork/-/merge_requests/97. There are many examples of ways to fix
failing tests.

System Tests
============

System tests for Stork are designed to test the software in a distributed environment.
They allow several Stork servers and agents running at the same time
to be tested in one test case, inside ``LXD`` containers. It is possible to set up
Kea services along with Stork agents. The framework enables experimentation
in containers, so custom Kea configurations can be deployed or specific Kea daemons
can be stopped.

The tests can use the Stork server RESTful API directly or the Stork web UI via Selenium.

Dependencies
------------
System tests require:

- Linux operating system (preferably Ubuntu or Fedora)
- Python 3
- ``LXD`` containers (https://linuxcontainers.org/lxd/introduction)

LXD Installation
----------------

The easiest way to install ``LXD`` is to use ``snap``. First, install ``snap``.

On Fedora:

.. code-block:: console

                $ sudo dnf install snapd

On Ubuntu:

.. code-block:: console

                $ sudo apt install snapd

Then install ``LXD``:

.. code-block:: console

                $ sudo snap install lxd

And then add the user to the ``lxd`` group:

.. code-block:: console

                $ sudo usermod -a -G lxd $USER

Log in again to make the user's presence in the ``lxd`` group visible in the shell session.

After installing ``LXD``, initialize it by running:

.. code-block:: console

                $ lxd init

and then for each question press **Enter**, i.e., use the default values::

   Would you like to use LXD clustering? (yes/no) [default=no]: **Enter**
   Do you want to configure a new storage pool? (yes/no) [default=yes]: **Enter**
   Name of the new storage pool [default=default]: **Enter**
   Name of the storage backend to use (dir, btrfs) [default=btrfs]: **Enter**
   Would you like to create a new btrfs subvolume under /var/snap/lxd/common/lxd? (yes/no) [default=yes]: **Enter**
   Would you like to connect to a MAAS server? (yes/no) [default=no]: **Enter**
   Would you like to create a new local network bridge? (yes/no) [default=yes]: **Enter**
   What should the new bridge be called? [default=lxdbr0]: **Enter**
   What IPv4 address should be used? (CIDR subnet notation, "auto" or "none") [default=auto]: **Enter**
   What IPv6 address should be used? (CIDR subnet notation, "auto" or "none") [default=auto]: **Enter**
   Would you like LXD to be available over the network? (yes/no) [default=no]: **Enter**
   Would you like stale cached images to be updated automatically? (yes/no) [default=yes] **Enter**
   Would you like a YAML "lxd init" preseed to be printed? (yes/no) [default=no]: **Enter**

More details can be found at: https://linuxcontainers.org/lxd/getting-started-cli/

The subvolume is stored in ``/var/snap/lxd/common/lxd``, and
is used to store images and containers. If the space is exhausted,
it is not possible to create new containers. This is not connected with total disk
space but rather with the space in this subvolume. To free space, remove stale images
or stopped containers. Basic usage of ``LXD`` is explained at:
https://linuxcontainers.org/lxd/getting-started-cli/#lxd-client

LXD Troubleshooting on Arch
~~~~~~~~~~~~~~~~~~~~~~~~~~~

**Problem**: After running ``lxd init``, an error message is returned:

.. code-block:: console

    Error: Failed to connect to local LXD: Get "http://unix.socket/1.0": dial unix /var/lib/lxd/unix.socket: connect: no such file or directory

**Solution**: Restart the ``lxd`` daemon:

.. code-block:: console

    sudo systemctl restart lxd

--------------

**Problem**: After running ``rake system_tests``, a message is displayed that ends in:

.. code-block:: console

    ************ START   tests.py::test_users_management[ubuntu/18.04-centos/7] **************************************************************

    stork-agent-ubuntu-18-04-gw0: {'fg': 'yellow', 'style': ''}
    stork-server-centos-7-gw0: {'fg': 'red', 'style': 'bold'}

But nothing else happens, and CPU and RAM usage by ``lxd`` are ~0%.

**Solution**: `See this original post <https://discuss.linuxcontainers.org/t/solved-arch-linux-containers-only-run-when-security-privileged-true/4006/5>`_

1. Create an ``/etc/subuid`` file with content:

.. code-block:: console

    root:1000000:65536

1. Create ``/etc/subgid`` with the same content.
2. Add these lines to ``/etc/default/lxc``:

.. code-block:: console

    lxc.idmap = u 0 100000 65536
    lxc.idmap = g 0 100000 65536


Running System Tests
--------------------

After preparing all the dependencies, the tests can be started;
however, the RPM and deb Stork packages need to be prepared first. This can
be done with this Rake task:

.. code-block:: console

                $ rake build_pkgs_in_docker

When using packages, the tests can be invoked by the following Rake task:

.. code-block:: console

                $ rake system_tests

This command first prepares the Python virtual environment (``venv``)
where ``pytest`` and other Python dependencies are installed. ``pytest`` is a Python testing
framework that is used in Stork system tests.

At the end of the logs are listed test cases with their result status.

The tests can be invoked directly using ``pytest``, but first the directory
must be changed to ``tests/system``:

.. code-block:: console

                $ cd tests/system
                $ ./venv/bin/pytest --tb=long -l -r ap -s tests.py

The switches passed to ``pytest`` are:

- ``--tb=long``: in case of failures, present the traceback in long format
- ``-l``: show values of local variables in tracebacks
- ``-r ap``: at the end of execution, print a report that includes (p)assed and (a)ll except passed (p)

To run a particular test case, add it just after ``test.py``:

.. code-block:: console

                $ ./venv/bin/pytest --tb=long -l -r ap -s tests.py::test_users_management[centos/7-ubuntu/18.04]

To get a list of tests without actually running them, the following command can be used:

.. code-block:: console

    $ ./venv/bin/pytest --collect-only tests.py

The names of all available tests are printed as ``<Function name_of_the_test>``.

A single test case can be run using a ``rake`` task with the test variable set to the test name:

.. code-block:: console

                $ rake system_tests test=tests.py::test_users_management[centos/7-ubuntu/18.04]


Developing System Tests
-----------------------

System tests are defined in ``tests.py`` and other files that start with ``test_``.
There are two other files that define the framework for Stork system tests:

- ``conftest.py`` - defines hooks for ``pytests``
- ``containers.py`` - handles LXD containers: starting/stopping; communication, such as
  invoking commands; uploading/downloading files; and installing and preparing the Stork
  agent/server and Kea and other dependencies that they require.

Most tests are constructed as follows:

.. code-block:: python

    @pytest.mark.parametrize("agent, server", SUPPORTED_DISTROS)
    def test_machines(agent, server):
        # login to stork server
        r = server.api_post('/sessions',
                            json=dict(useremail='admin', userpassword='admin'),
                            expected_status=200)
        assert r.json()['login'] == 'admin'

        # add machine
        machine = dict(
            address=agent.mgmt_ip,
            agentPort=8080)
        r = server.api_post('/machines', json=machine, expected_status=200)
        assert r.json()['address'] == agent.mgmt_ip

        # wait for application discovery by Stork Agent
        for i in range(20):
            r = server.api_get('/machines')
            data = r.json()
            if len(data['items']) == 1 and \
               len(data['items'][0]['apps'][0]['details']['daemons']) > 1:
                break
            time.sleep(2)

        # check discovered application by Stork Agent
        m = data['items'][0]
        assert m['apps'][0]['version'] == '1.7.3'

It may be useful to explain each part of this code.

.. code-block:: python

    @pytest.mark.parametrize("agent, server", SUPPORTED_DISTROS)

This indicates that the test is parameterized: there will be one or more
instances of this test in execution for each set of parameters.

The constant ``SUPPORTED_DISTROS`` defines two sets of operating systems
for testing:

.. code-block:: python

    SUPPORTED_DISTROS = [
        ('ubuntu/18.04', 'centos/7'),
        ('centos/7', 'ubuntu/18.04')
    ]

The first set indicates that for the Stork agent, ``Ubuntu 18.04`` should be used
in the LXD container, and for the Stork server, ``CentOS 7``. The second set is the opposite
of the first one.

The next line:

.. code-block:: python

    def test_machines(agent, server):

defines the test function. Normally, the agent and server argument would get the text values
``'ubuntu/18.04'`` and ``'centos/7'``, but a hook exists in the ``pytest_pyfunc_call()`` function
of ``conftest.py`` that intercepts these arguments and
uses them to spin up LXD containers with the indicated operating systems. This hook
also collects Stork logs from these containers at the end of the test and stores
them in the ``test-results`` folder for later analysis if needed.

Instead of text values, the hook replaces the arguments with references
to actual LXC container objects, so that the test can interact directly with them.
Besides substituting the ``agent`` and ``server`` arguments, the hook intercepts
any argument that starts with ``agent`` or ``server``. This allows
multiple agents in the test, e.g. ``agent1``, ``agent_kea``, or ``agent_bind9``.

Next, log into the Stork server using its REST API:

.. code-block:: python

        # login to stork server
        r = server.api_post('/sessions',
                            json=dict(useremail='admin', userpassword='admin'),
                            expected_status=200)
        assert r.json()['login'] == 'admin'

Then, add a machine with a Stork agent to the Stork server:

.. code-block:: python

        # add machine
        machine = dict(
            address=agent.mgmt_ip,
            agentPort=8080)
        r = server.api_post('/machines', json=machine, expected_status=200)
        assert r.json()['address'] == agent.mgmt_ip

A check then verifies the returned address of the machine.

After a few seconds, ``stork-agent`` detects the Kea application and reports it
to ``stork-server``. The server is periodically polled for updated
information about the Kea application.

.. code-block:: python

        # wait for application discovery by Stork Agent
        for i in range(20):
            r = server.api_get('/machines')
            data = r.json()
            if len(data['items']) == 1 and \
               len(data['items'][0]['apps'][0]['details']['daemons']) > 1:
                break
            time.sleep(2)

Finally, the returned data about Kea can be verified:

.. code-block:: python

        # check discovered application by Stork Agent
        m = data['items'][0]
        assert m['apps'][0]['version'] == '1.7.3'

.. _docker_containers_for_development:

Docker Containers for Development
=================================

To ease development, there are several Docker containers available.
These containers are used in the Stork demo and are fully
described in the :ref:`Demo` chapter.

The following ``Rake`` tasks start these containers.

.. table:: Rake tasks for managing development containers
   :class: longtable
   :widths: 26 74

   +----------------------------------------+---------------------------------------------------------------+
   | Rake Task                              | Description                                                   |
   +========================================+===============================================================+
   | ``rake build_kea_container``           | Build a container ``agent-kea`` with a Stork agent            |
   |                                        | and Kea with DHCPv4.                                          |
   +----------------------------------------+---------------------------------------------------------------+
   | ``rake run_kea_container``             | Start an ``agent-kea`` container. Published port is 8888.     |
   +----------------------------------------+---------------------------------------------------------------+
   | ``rake build_kea6_container``          | Build an ``agent-kea6`` container with a Stork agent          |
   |                                        | and Kea with DHCPv6.                                          |
   +----------------------------------------+---------------------------------------------------------------+
   | ``rake run_kea6_container``            | Start an ``agent-kea6`` container. Published port is 8886.    |
   +----------------------------------------+---------------------------------------------------------------+
   | ``rake build_kea_ha_containers``       | Build two containers, ``agent-kea-ha1`` and ``agent-kea-ha2``,|
   |                                        | that are configured to work together in High                  |
   |                                        | Availability mode, with Stork agents, and Kea with DHCPv4.    |
   +----------------------------------------+---------------------------------------------------------------+
   | ``rake run_kea_ha_containers``         | Start the ``agent-kea-ha1`` and ``agent-kea-ha2`` containers. |
   |                                        | Published ports are 8881 and 8882.                            |
   +----------------------------------------+---------------------------------------------------------------+
   | ``rake build_kea_premium_container``   | Build an ``agent-kea-premium`` container with a Stork agent   |
   |                                        | and Kea with DHCPv4 with host reservations stored in          |
   |                                        | a database. This requires **premium** features.               |
   +----------------------------------------+---------------------------------------------------------------+
   | ``rake run_kea_premium_container``     | Start the ``agent-kea-premium`` container. This requires      |
   |                                        | **premium** features.                                         |
   +----------------------------------------+---------------------------------------------------------------+
   | ``rake build_bind9_container``         | Build an ``agent-bind9`` container with a Stork agent         |
   |                                        | and BIND 9.                                                   |
   +----------------------------------------+---------------------------------------------------------------+
   | ``rake run_bind9_container``           | Start an ``agent-bind9`` container. Published port is 9999.   |
   +----------------------------------------+---------------------------------------------------------------+

.. note::

    It is recommended that these commands be run using a user account without
    superuser privileges, which may require some previous steps to set up. On
    most systems, adding the account to the ``docker`` group should be enough.
    On most Linux systems, this is done with:

    .. code:: console

        $ sudo usermod -aG docker ${user}

    A restart may be required for the change to take effect.


Packaging
=========

There are scripts for packaging the binary form of Stork. There are
two supported formats: RPM and deb.

The RPM package is built on the latest CentOS version. The deb package
is built on the latest Ubuntu LTS.

There are two packages built for each system: a server and an agent.

Rake tasks can perform the entire build procedure in a
Docker container: ``build_rpms_in_docker`` and
``build_debs_in_docker``. It is also possible to build packages directly
in the current operating system; this is provided by the ``deb_agent``,
``rpm_agent``, ``deb_server``, and ``rpm_server`` Rake tasks.

Internally, these packages are built by FPM
(https://fpm.readthedocs.io/). The containers that are used to build
packages are prebuilt with all dependencies required, using the
``build_fpm_containers`` Rake task. The definitions
of these containers are placed in ``docker/pkgs/centos-8.txt`` and
``docker/pkgs/ubuntu-18-04.txt``.

Implementation details
======================

Agent Registration Process
--------------------------

The diagram below shows a flowchart of the agent registration process in Stork.
It merely demonstrates the successful registration path.
The first Certificate Signing Request (CSR) is generated using an existing or new
private key and agent token.
The CSR, server token (optional), and agent token are sent to the Stork server.
A successful server response contains a signed agent certificate, a server CA
certificate, and an assigned Machine ID.
If the agent was already registered with the provided agent token, only the assigned
machine ID is returned without new certificates.
The agent uses the returned machine ID to verify that the registration was successful.

.. figure:: uml/registration-agent.*

    Agent registration
