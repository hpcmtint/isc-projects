from datetime import datetime
import re
from typing import Callable
from pytest import param
import requests

from core.compose import DockerCompose
from core.utils import NoSuccessException, wait_for_success
from core.wrappers.base import ComposeServiceWrapper
import core.wrappers.api_types as api
from core.log_parser import GoLogEntry, split_log_messages


class UnexpectedStatusCodeException(Exception):
    def __init__(self, expected, actual) -> None:
        super().__init__("Unexpected HTTP status. Expected: %d, got: %d"
                         % (expected, actual))


class UnexpectedEventException(Exception):
    def __init__(self, event: api.Event):
        super().__init__("Unexpcted event occurs: %s" % event)


class Server(ComposeServiceWrapper):
    def __init__(self, compose: DockerCompose, service_name: str):
        super().__init__(compose, service_name)
        self._session = requests.Session()
        self._address = self._compose.get_service_ip_address(
            self._service_name, "storknet"
        )
        self._port = 8080

    def _fetch_api(self, method: str, endpoint: str,
                   expected_status: int = None, **kwargs):
        url = "http://%s:%s/api/%s" % (self._address, self._port, endpoint)
        r = self._session.request(method, url, **kwargs)
        if expected_status is not None and expected_status != r.status_code:
            raise UnexpectedStatusCodeException(expected_status, r.status_code)
        return r

    @property
    def ip_address(self):
        return self._address

    # Authorize

    def log_in(self, username: str, password: str) -> api.User:
        r = self._fetch_api("POST", "/sessions", expected_status=200,
                            json=dict(useremail=username, userpassword=password))
        return r.json()

    def log_in_as_admin(self):
        return self.log_in("admin", "admin")

    # List / Search

    def list_users(self, limit=10, offset=0) -> api.UserList:
        r = self._fetch_api("GET", "/users", expected_status=200,
                            params=dict(start=offset, limit=limit))
        return r.json()

    def list_groups(self, limit=10, offset=0) -> api.GroupList:
        r = self._fetch_api("GET", "/groups", expected_status=200,
                            params=dict(start=offset, limit=limit))
        return r.json()

    def list_machines(self, authorized=None, limit=10, offset=0) -> api.MachineList:
        params = dict(start=offset, limit=limit)
        if authorized is not None:
            params["authorized"] = str(authorized).lower()
        r = self._fetch_api("GET", "/machines", expected_status=200,
                            params=params)
        return r.json()

    def list_subnets(self, app_id=None, family: int = None, limit=10, offset=0) -> api.SubnetList:
        params = dict(start=offset, limit=limit)
        if app_id is not None:
            params["appID"] = app_id
        if family is not None:
            params["dhcpVersion"] = family

        r = self._fetch_api("GET", "/subnets", expected_status=200,
                            params=params)
        return r.json()

    def list_events(self, daemon_type=None, app_type=None, machine_id=None,
                    user_id=None, limit=10, offset=0) -> api.EventList:
        params = dict(start=offset, limit=limit)
        if daemon_type is not None:
            params["daemonType"] = daemon_type
        if app_type is not None:
            params["appType"] = app_type
        if machine_id is not None:
            params["machine"] = machine_id
        if user_id is not None:
            params["user"] = user_id

        r = self._fetch_api("GET", "/events", expected_status=200,
                            params=params)
        return r.json()

    # Create

    def create_user(self, user_create: api.UserCreate):
        self._fetch_api("POST", "/users", json=user_create,
                        expected_status=200)

    # Read

    def read_machine_state(self, machine_id: int) -> api.MachineState:
        r = self._fetch_api("GET", "/machines/%d/state" % machine_id,
                            expected_status=200)
        return r.json()

    # Update

    def update_machine(self, machine: api.Machine) -> api.Machine:
        r = self._fetch_api("PUT", '/machines/%d' % machine['id'],
                            json=machine, expected_status=200)
        return r.json()

    # Complex

    def authorize_all_machines(self) -> api.MachineList:
        machines = self.list_machines(False)
        machine: api.Machine
        for machine in machines["items"]:
            machine["authorized"] = True
            self.update_machine(machine)
        return machines

    def _search_for_logs(self, condition: Callable[[GoLogEntry], bool]):
        logs, _ = self._compose.get_logs(self._service_name)
        for entry in split_log_messages(logs):
            go_entry = entry.as_go()
            if condition(go_entry):
                yield go_entry

    # Waits

    @wait_for_success(UnexpectedStatusCodeException)
    def _wait_for_success_response(self, request, *args, **kwargs):
        return request(self, *args, **kwargs)

    def _wait_for_event(self,
                        expected_condition: Callable[[api.Event], bool] = None,
                        unexpected_condition: Callable[[
                            api.Event], bool] = None,
                        **kwargs):
        fetch_timestamp = datetime.min

        @wait_for_success(NoSuccessException)
        def worker():
            nonlocal fetch_timestamp

            events = self.list_events(limit=100, **kwargs)
            for event in reversed(events["items"]):
                # skip older events
                timestamp = datetime.fromisoformat(
                    event["createdAt"].rstrip("Z"))
                if timestamp < fetch_timestamp:
                    continue
                fetch_timestamp = timestamp

                if expected_condition is not None and expected_condition(event):
                    return
                if unexpected_condition is not None and unexpected_condition(event):
                    raise UnexpectedEventException(event)
            if expected_condition is not None:
                raise NoSuccessException()

        return worker()

    def wait_for_next_machine_state(self, machine_id: int) -> api.MachineState:
        start = datetime.utcnow()

        @wait_for_success()
        def worker():
            state = self.read_machine_state(machine_id)
            last_visited = datetime.fromisoformat(
                state["lastVisitedAt"].rstrip("Z"))
            if last_visited < start:
                raise NoSuccessException()
            return state
        return worker()

    _pattern_added_subnets = re.compile(
        r'added (?:(?:\d+ subnets)|(?:<subnet.*>)) to <daemon '
        r'id="(?P<daemon_id>\d+)" '
        r'name="(?P<daemon_name>.*)" '
        r'appId="(?P<app_id>\d+)"')

    def wait_for_adding_subnets(self, daemon_id: int = None, daemon_name: str = None, app_id: int = None):
        def condition(ev: api.Event):
            match = Server._pattern_added_subnets.search(ev["text"])
            if match is None:
                return False
            if daemon_id is not None and match.group("daemon_id") != str(daemon_id):
                return False
            if daemon_name is not None and match.group("daemon_name") != daemon_name:
                return False
            if app_id is not None and match.group("app_id") != str(app_id):
                return False
            return True

        self._wait_for_event(condition)
