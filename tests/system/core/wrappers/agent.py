from typing import Callable, Generator, List
import urllib3

from core.compose import DockerCompose
from core.wrappers.base import ComposeServiceWrapper
from core.wrappers.server import Server
from core.utils import memoize, wait_for_success, NoSuccessException
from core.prometheus_parser import text_fd_to_metric_families, Metric


class Agent(ComposeServiceWrapper):
    """A wrapper for the Stork Agent docker-compose service."""

    def __init__(self, compose: DockerCompose, service_name: str,
                 server_service: Server):
        """
        A wrapper constructor.

        Parameters
        ----------
        compose : DockerCompose
            The compose controller object
        service_name : str
            The name of the docker-compose service containing the Stork Agent.
        server_service : Server
            The wrapper for the Stork Server service where this agent is
            registered. If the registration was suppressed then it should be
            a None value.
        """
        super().__init__(compose, service_name)
        self._server_service = server_service

    @memoize
    def _get_metrics_endpoint(self):
        """Returns URL of the agent metrics endpoint."""
        internal_port = 9119
        mapped = self._compose.port(self._service_name, internal_port)
        url = "http://%s:%d/metrics" % mapped
        return url

    @property
    def server(self):
        """Returns a Server wrapper where this agent is registered. If the
        registration was suppressed then it returns None."""
        return self._server_service

    def hash_cert_files(self):
        """Calculates the hashes of the TLS credentials used by the agent."""
        cert_paths = [
            '/var/lib/stork-agent/certs/key.pem',
            '/var/lib/stork-agent/certs/cert.pem',
            '/var/lib/stork-agent/certs/ca.pem',
            '/var/lib/stork-agent/tokens/agent-token.txt'
        ]

        hashes = {}
        for cert_path in cert_paths:
            hash_ = self._hash_file(cert_path)
            hashes[cert_path] = hash_
        return hashes

    def read_prometheus_metrics(self):
        """Reads the Prometheus metrics collected by the Stork Agent."""
        url = self._get_metrics_endpoint()
        http = urllib3.PoolManager()
        resp = http.request("GET", url)
        data = resp.data.decode("utf-8")
        data = data.split("\n")

        return text_fd_to_metric_families(data)

    def restart_stork_agent(self):
        """
        Restarts the Stork Agent and waits to recover an operational status.
        """
        self._restart_supervisor_service('stork-agent')

    def reload_stork_agent(self):
        """Sends SIGHUP to the stork-agent."""
        self._reload_supervisor_service('stork-agent')

    def is_registered(self):
        """True if an agent was successfully registered. Otherwise False."""
        if self._server_service is None:
            return False
        # ToDo: Using logs is a little dangerous. They can contain a bloat data.
        stdout, _ = self._compose.logs()
        return "machine registered" in stdout.lower()

    @wait_for_success(wait_msg="Waiting to be registered...")
    def wait_for_registration(self):
        """Block the execution until registration passes."""
        if not self.is_registered():
            raise NoSuccessException()


    def get_stork_agent_pid(self):
        """Returns PID of the stork-agent process."""
        return self._get_pid('stork-agent')
