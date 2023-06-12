from core.wrappers import Server
from core.fixtures import server_parametrize


@server_parametrize("server-ldap")
def test_run_server_with_ldap_hook(server_service: Server):
    authentication_methods = server_service.list_authentication_methods()
    assert len(authentication_methods) == 2

    server_service.log_in_as_admin()
