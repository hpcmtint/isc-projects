.. _troubleshooting:

***************
Troubleshooting
***************

``stork-agent``
===============

This section describes the solutions for some common issues with the Stork agent.

--------------

:Issue:       A machine is authorized in the Stork server successfully, but there are no applications.
:Description: The user installed and started ``stork-server`` and ``stork-agent`` and authorized
              the machine. The "Last Refreshed" column has a value on the Machines page, the
              "Error" column value shows no errors, but the "Daemons" column is still blank.
              The "Application" section on the specific Machine page is also blank.
:Solution:    Make sure that the daemons are running:

              - Kea Control Agent, Kea DHCPv4 server, and/or Kea DHCPv6 server
              - BIND 9
:Explanation: If the "Last Refreshed" column has a value, and the "Error" column value has no errors,
              the communication between ``stork-server`` and ``stork-agent`` works correctly, which implies that
              the cause of the problem is between the Stork agent and the daemons. The most likely issue is that none of
              the Kea/BIND 9 daemons are running. ``stork-agent`` communicates with the BIND 9 daemon
              directly; however, it communicates with the Kea DHCPv4 and Kea DHCPv6 servers via the
              Kea Control Agent. If only the "CA" daemon is displayed in the Stork interface, the Kea Control Agent
              is running, but the DHCP daemons are not.

--------------

:Issue:       After starting the Stork agent, it gets stuck in an infinite "sleeping" loop.
:Description: ``stork-agent`` is running with server support (the ``--listen-prometheus-only`` flag is unused).
              The ``try to register agent in Stork server`` message is displayed initially, but the agent only
              prints the recurring ``sleeping for 10 seconds before next registration attempt`` message.
:Solution 1.: ``stork-server`` is not running. Start the Stork server first and restart the ``stork-agent`` daemon.
:Solution 2.: The configured server URL in ``stork-agent`` is invalid. Correct the URL and restart the agent.

--------------

:Issue:       After starting ``stork-agent``, it keeps printing the following messages:
              ``loaded server cert: /var/lib/stork-agent/certs/cert.pem and key: /var/lib/stork-agent/certs/key.pem``
:Description: ``stork-agent`` runs correctly, and its registration is successful.
              After the ``started serving Stork Agent`` message, the agent prints the recurring message about loading server certs.
              The network traffic analysis to the server reveals that it rejects all packets from the agent
              (TLS HELLO handshake failed).
:Solution:    Re-register the agent to regenerate the certificates, using the ``stork-agent register`` command.
:Explanation: The ``/var/lib/stork-agent/certs/ca.pem`` file is missing or corrupted.
              The re-registration removes old files and creates new ones.

--------------

:Issue:       The cert PEM file is not loaded.
:Description: The agent fails to start and prints an ``open /var/lib/stork-agent/certs/cert.pem: no such file or directory
              could not load cert PEM file: /var/lib/stork-agent/certs/cert.pem`` error message.
:Solution:    Re-register the agent to regenerate the certificates, using the ``stork-agent register`` command.

--------------

:Issue:       A connection problem to the DHCP daemon(s).
:Description: The agent prints the message ``problem with connecting to dhcp daemon: unable to forward command to
              the dhcp6 service: No such file or directory. The server is likely to be offline``.
:Solution:    Try to start the Kea service: ``systemctl start kea-dhcp4 kea-dhcp6``
:Explanation: The ``kea-dhcp4.service`` or ``kea-dhcp6.service`` (depending on the service type in the message) is not running.
              If the above commands do not resolve the problem, check the Kea Administrator Reference
              Manual (ARM) for troubleshooting assistance.

--------------

:Issue:       ``stork-agent`` receives a ``remote error: tls: certificate required`` message from the Kea Control Agent.
:Description: The Stork agent and the Kea Control Agent are running, but they cannot establish a connection.
              The ``stork-agent`` log contains the error message mentioned above.
:Solution:    Install the valid TLS certificates in ``stork-agent`` or set the ``cert-required`` value in ``/etc/kea/kea-ctrl-agent.conf`` to ``false``.
:Explanation: By default, ``stork-agent`` does not use TLS when it connects to Kea. If the Kea Control Agent configuration
              includes the ``cert-required`` value set to ``true``, it requires the Stork agent to use secure connections
              with valid, trusted TLS certificates. It can be turned off by setting the ``cert-required`` value to
              ``false`` when using self-signed certificates, or the Stork agent TLS credentials
              can be replaced with trusted ones.

--------------

:Issue:       Kea Control Agent returns a ``Kea error response - status: 401, message: Unauthorized`` message.
:Description: The Stork agent and the Kea Control Agent are running, but they cannot connect.
              The ``stork-agent`` logs contain similar messages: ``failed to parse responses from Kea:
              { "result": 401, "text": "Unauthorized" }`` or ``Kea error response - status: 401, message: Unauthorized``.
:Solution:    Update the ``/etc/stork/agent-credentials.json`` file with the valid user/password credentials.
:Explanation: The Kea Control Agent can be configured to use Basic Authentication. If it is enabled,
              valid credentials must be provided in the ``stork-agent`` configuration. Verify that this file exists
              and contains a valid username, password, and IP address.

--------------

:Issue:       During the registration process, ``stork-agent`` prints a ``problem with registering machine:
              cannot parse address`` message.
:Description: Stork is configured to use an IPv6 link-local address. The agent prints the
              ``try to register agent in Stork server`` message and then the above error. The agent exists
              with a fatal status.
:Solution:    Use a global IPv6 or an IPv4 address.
:Explanation: IPv6 link-local addresses are not supported by ``stork-server``.

--------------

:Issue:       A protocol problem occurs during the agent registration.
:Description: During the registration process, ``stork-agent`` prints a ``problem with registering machine:
              Post "/api/machines": unsupported protocol scheme ""`` message.
:Solution:    The ``--server-url`` argument is provided in the wrong format; it must be a canonical URL.
              It should begin with the protocol (``http://`` or ``https://``), contain the host (DNS name or
              IP address; for IPv6 escape them with square brackets), and end with the port
              (delimited from the host by a colon). For example: ``http://storkserver:8080``.

---------------

:Issue:       The values in ``/etc/stork/agent.env`` or ``/etc/stork/agent-credentials.json`` were changed,
              but ``stork-agent`` does not noticed the changes.
:Solution 1.: Restart the daemon.
:Solution 2.: Send the SIGHUP signal to the ``stork-agent`` process.
:Explanation: ``stork-agent`` reads configurations at startup or after receiving the SIGHUP signal.

--------------

:Issue:       The values in ``/etc/stork/agent.env`` were changed and the Stork agent was restarted, but
              it still uses the default values.
:Description: The agent is running using the ``stork-agent`` command. It uses the parameters passed
              from the command line but ignores the ``/etc/stork/agent.env`` file entries.
              If the agent is running as the systemd daemon, it uses the expected values.
:Solution 1.: Load the environment variables from the ``/etc/stork/agent.env`` file before running Stork agent.
              For example, run ``. /etc/stork/agent.env``.
:Solution 2.: Run the Stork agent with the ``--use-env-file`` switch.
:Explanation: The ``/etc/stork/agent.env`` file contains the environment variables, but ``stork-agent`` does not automatically
              load them, unless you use ``--use-env-file flag``; the file must be loaded manually. The default systemd service
              unit is configured to load this file before starting the agent.

--------------

:Issue:       Stork agent doesn't start with the following error:
              ``Cannot start the Stork Agent: plugin.Open("[HOOK DIRECTORY]/[FILENAME]"): [HOOK DIRECTORY]/[FILENAME]: file too short`` or 
              ``Cannot start the Stork Agent: plugin.Open("[HOOK DIRECTORY]/[FILENAME]"): [HOOK DIRECTORY]/[FILENAME]: invalid ELF header``
:Solution:    Remove the given file from the hook directory.
:Explanation: The file under a given path is not valid Stork hook.

--------------

:Issue:       Stork agent doesn't start with the following error:
              ``Cannot start the Stork Agent: incompatible hook version: 1.0.0``
:Solution:    Update the given hook.
:Explanation: The hook is out-of-date. It's incompatible with the Stork core
              application.

--------------

:Issue:       Stork agent doesn't start with the following error:
              ``Cannot start the Stork Agent: plugin: symbol Version not found in plugin``
:Solution:    Remove or fix the given file.
:Explanation: Hook directory contains Go plugin but that is not a hook; Hook
              doesn't contain required symbol.

--------------

:Issue:       Stork agent doesn't start with the following error:
              ``Cannot start the Stork Agent: hook library dedicated for another program: Stork Server``
:Solution:    Move the incompatible hooks to a separate directory.

--------------

:Issue:       Stork agent starts but the hooks aren't loaded. The logs comprise
              the following message:
              ``Cannot find plugin paths in: /var/lib/stork-agent/hooks: cannot list hook directory: /var/lib/stork-agent/hooks: open /var/lib/stork-agent/hooks: no such file or directory``
:Solution:    Create the hook directory or change the path in the configuration.
:Explanation: Hook directory doesn't exist.

--------------

:Issue:       Stork agent doesn't start with the following error:
              ``Cannot start the Stork Agent: open [HOOK DIRECTORY]: permission denied cannot list hook directory``
:Solution:    Grant the right for read the hook directory for the Stork user.
:Explanation: The hook directory is not readable.

--------------

:Issue:       Stork agent doesn't start with the following error:
              ``Cannot start the Stork Agent: readdirent [HOOK DIRECTORY]/[FILENAME]: not a directory cannot list hook directory``
:Solution:    Change the hook directory path.
:Explanation: Directory is a file.

``stork-server``
================

This section describes the solutions for some common issues with the Stork server.

---------------

:Issue:       The values in ``/etc/stork/server.env`` were changed,
              but ``stork-server`` does not noticed the changes.
:Solution 1.: Restart the daemon.
:Solution 2.: Send the SIGHUP signal to the ``stork-server`` process.
:Explanation: ``stork-server`` reads configurations at startup or after receiving the SIGHUP signal.

--------------

:Issue:       The values in ``/etc/stork/server.env`` were changed and the Stork server was restarted, but
              it still uses the default values.
:Description: The server is running using the ``stork-server`` command. It uses the parameters passed
              from the command line but ignores the ``/etc/stork/server.env`` file entries.
              If the server is running as the systemd daemon, it uses the expected values.
:Solution 1.: Load the environment variables from the ``/etc/stork/server.env`` file before running Stork server.
              For example, run ``. /etc/stork/server.env``.
:Solution 2.: Run the Stork server with the ``--use-env-file`` switch.
:Explanation: The ``/etc/stork/server.env`` file contains the environment variables, but ``stork-server`` does not automatically
              load them, unless you use ``--use-env-file`` flag; the file must be loaded manually. The default systemd service
              unit is configured to load this file before starting the server.

--------------

:Issue:       Stork server doesn't start with the following error:
              ``Cannot start the Stork Server: plugin.Open("[HOOK DIRECTORY]/[FILENAME]"): [HOOK DIRECTORY]/[FILENAME]: file too short`` or 
              ``Cannot start the Stork Server: plugin.Open("[HOOK DIRECTORY]/[FILENAME]"): [HOOK DIRECTORY]/[FILENAME]: invalid ELF header``
:Solution:    Remove the given file from the hook directory.
:Explanation: The file under a given path is not valid Stork hook.

--------------

:Issue:       Stork server doesn't start with the following error:
              ``Cannot start the Stork Server: incompatible hook version: 1.0.0``
:Solution:    Update the given hook.
:Explanation: The hook is out-of-date. It's incompatible with the Stork core
              application.

--------------

:Issue:       Stork server doesn't start with the following error:
              ``Cannot start the Stork Server: plugin: symbol Version not found in plugin``
:Solution:    Remove or fix the given file.
:Explanation: Hook directory contains Go plugin but that is not a hook; Hook
              doesn't contain required symbol.

--------------

:Issue:       Stork server doesn't start with the following error:
              ``Cannot start the Stork Server: hook library dedicated for another program: Stork Agent``
:Solution:    Move the incompatible hooks to a separate directory.

--------------

:Issue:       Stork server starts but the hooks aren't loaded. The logs comprise
              the following message:
              ``Cannot find plugin paths in: /var/lib/stork-server/hooks: cannot list hook directory: /var/lib/stork-server/hooks: open /var/lib/stork-server/hooks: no such file or directory``
:Solution:    Create the hook directory or change the path in the configuration.
:Explanation: Hook directory doesn't exist.

--------------

:Issue:       Stork server doesn't start with the following error:
              ``Cannot start the Stork Server: open [HOOK DIRECTORY]: permission denied cannot list hook directory``
:Solution:    Grant the right for read the hook directory for the Stork user.
:Explanation: The hook directory is not readable.

--------------

:Issue:       Stork server doesn't start with the following error:
              ``Cannot start the Stork Server: readdirent [HOOK DIRECTORY]/[FILENAME]: not a directory cannot list hook directory``
:Solution:    Change the hook directory path.
:Explanation: Directory is a file.