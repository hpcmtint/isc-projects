// Code generated by stork-code-gen. DO NOT EDIT.

/**
 * Attention! Generated Code!
 *
 * Run "rake gen:std_option_defs" to regenerate the option definitions
 * specified in the "codegen/std_dhcpv4_option_def.json", using the
 * template file "std-dhcpv4-option-defs.ts.template", into the
 * "std-dhcpv4-option-defs.ts".
 */

/**
 * Standard DHCPv4 option definitions.
 */
export const stdDhcpv4OptionDefs = [
    {
        code: 1,
        name: 'subnet-mask',
        type: 'ipv4-address',
        space: 'dhcp4',
    },
    {
        code: 2,
        name: 'time-offset',
        type: 'int32',
        space: 'dhcp4',
    },
    {
        code: 3,
        name: 'routers',
        type: 'ipv4-address',
        space: 'dhcp4',
    },
    {
        code: 4,
        name: 'time-servers',
        type: 'ipv4-address',
        space: 'dhcp4',
    },
    {
        code: 5,
        name: 'name-servers',
        type: 'ipv4-address',
        space: 'dhcp4',
    },
    {
        code: 6,
        name: 'domain-name-servers',
        type: 'ipv4-address',
        space: 'dhcp4',
    },
    {
        code: 7,
        name: 'log-servers',
        type: 'ipv4-address',
        space: 'dhcp4',
    },
    {
        code: 8,
        name: 'cookie-servers',
        type: 'ipv4-address',
        space: 'dhcp4',
    },
    {
        code: 9,
        name: 'lpr-servers',
        type: 'ipv4-address',
        space: 'dhcp4',
    },
    {
        code: 10,
        name: 'impress-servers',
        type: 'ipv4-address',
        space: 'dhcp4',
    },
    {
        code: 11,
        name: 'resource-location-servers',
        type: 'ipv4-address',
        space: 'dhcp4',
    },
    {
        code: 12,
        name: 'host-name',
        type: 'string',
        space: 'dhcp4',
    },
    {
        code: 13,
        name: 'boot-size',
        type: 'uint16',
        space: 'dhcp4',
    },
    {
        code: 14,
        name: 'merit-dump',
        type: 'string',
        space: 'dhcp4',
    },
    {
        code: 15,
        name: 'domain-name',
        type: 'string',
        space: 'dhcp4',
    },
    {
        code: 16,
        name: 'swap-server',
        type: 'ipv4-address',
        space: 'dhcp4',
    },
    {
        code: 17,
        name: 'root-path',
        type: 'string',
        space: 'dhcp4',
    },
    {
        code: 18,
        name: 'extensions-path',
        type: 'string',
        space: 'dhcp4',
    },
    {
        code: 19,
        name: 'ip-forwarding',
        type: 'bool',
        space: 'dhcp4',
    },
    {
        code: 20,
        name: 'non-local-source-routing',
        type: 'bool',
        space: 'dhcp4',
    },
    {
        code: 21,
        name: 'policy-filter',
        type: 'ipv4-address',
        space: 'dhcp4',
    },
    {
        code: 22,
        name: 'max-dgram-reassembly',
        type: 'uint16',
        space: 'dhcp4',
    },
    {
        code: 23,
        name: 'default-ip-ttl',
        type: 'uint8',
        space: 'dhcp4',
    },
    {
        code: 24,
        name: 'path-mtu-aging-timeout',
        type: 'uint32',
        space: 'dhcp4',
    },
    {
        code: 25,
        name: 'path-mtu-plateau-table',
        type: 'uint16',
        space: 'dhcp4',
    },
    {
        code: 26,
        name: 'interface-mtu',
        type: 'uint16',
        space: 'dhcp4',
    },
    {
        code: 27,
        name: 'all-subnets-local',
        type: 'bool',
        space: 'dhcp4',
    },
    {
        code: 28,
        name: 'broadcast-address',
        type: 'ipv4-address',
        space: 'dhcp4',
    },
    {
        code: 29,
        name: 'perform-mask-discovery',
        type: 'bool',
        space: 'dhcp4',
    },
    {
        code: 30,
        name: 'mask-supplier',
        type: 'bool',
        space: 'dhcp4',
    },
    {
        code: 31,
        name: 'router-discovery',
        type: 'bool',
        space: 'dhcp4',
    },
    {
        code: 32,
        name: 'router-solicitation-address',
        type: 'ipv4-address',
        space: 'dhcp4',
    },
    {
        code: 33,
        name: 'static-routes',
        type: 'ipv4-address',
        space: 'dhcp4',
    },
    {
        code: 34,
        name: 'trailer-encapsulation',
        type: 'bool',
        space: 'dhcp4',
    },
    {
        code: 35,
        name: 'arp-cache-timeout',
        type: 'uint32',
        space: 'dhcp4',
    },
    {
        code: 36,
        name: 'ieee802-3-encapsulation',
        type: 'bool',
        space: 'dhcp4',
    },
    {
        code: 37,
        name: 'default-tcp-ttl',
        type: 'uint8',
        space: 'dhcp4',
    },
    {
        code: 38,
        name: 'tcp-keepalive-interval',
        type: 'uint32',
        space: 'dhcp4',
    },
    {
        code: 39,
        name: 'tcp-keepalive-garbage',
        type: 'bool',
        space: 'dhcp4',
    },
    {
        code: 40,
        name: 'nis-domain',
        type: 'string',
        space: 'dhcp4',
    },
    {
        code: 41,
        name: 'nis-servers',
        type: 'ipv4-address',
        space: 'dhcp4',
    },
    {
        code: 42,
        name: 'ntp-servers',
        type: 'ipv4-address',
        space: 'dhcp4',
    },
    {
        code: 43,
        name: 'vendor-encapsulated-options',
        type: 'empty',
        space: 'dhcp4',
        encapsulate: 'vendor-encapsulated-options-space',
    },
    {
        code: 44,
        name: 'netbios-name-servers',
        type: 'ipv4-address',
        space: 'dhcp4',
    },
    {
        code: 45,
        name: 'netbios-dd-server',
        type: 'ipv4-address',
        space: 'dhcp4',
    },
    {
        code: 46,
        name: 'netbios-node-type',
        type: 'uint8',
        space: 'dhcp4',
    },
    {
        code: 47,
        name: 'netbios-scope',
        type: 'string',
        space: 'dhcp4',
    },
    {
        code: 48,
        name: 'font-servers',
        type: 'ipv4-address',
        space: 'dhcp4',
    },
    {
        code: 49,
        name: 'x-display-manager',
        type: 'ipv4-address',
        space: 'dhcp4',
    },
    {
        code: 50,
        name: 'dhcp-requested-address',
        type: 'ipv4-address',
        space: 'dhcp4',
    },
    {
        code: 51,
        name: 'dhcp-lease-time',
        type: 'uint32',
        space: 'dhcp4',
    },
    {
        code: 52,
        name: 'dhcp-option-overload',
        type: 'uint8',
        space: 'dhcp4',
    },
    {
        code: 53,
        name: 'dhcp-message-type',
        type: 'uint8',
        space: 'dhcp4',
    },
    {
        code: 54,
        name: 'dhcp-server-identifier',
        type: 'ipv4-address',
        space: 'dhcp4',
    },
    {
        code: 55,
        name: 'dhcp-parameter-request-list',
        type: 'uint8',
        space: 'dhcp4',
    },
    {
        code: 56,
        name: 'dhcp-message',
        type: 'string',
        space: 'dhcp4',
    },
    {
        code: 57,
        name: 'dhcp-max-message-size',
        type: 'uint16',
        space: 'dhcp4',
    },
    {
        code: 58,
        name: 'dhcp-renewal-time',
        type: 'uint32',
        space: 'dhcp4',
    },
    {
        code: 59,
        name: 'dhcp-rebinding-time',
        type: 'uint32',
        space: 'dhcp4',
    },
    {
        code: 60,
        name: 'vendor-class-identifier',
        type: 'string',
        space: 'dhcp4',
    },
    {
        code: 61,
        name: 'dhcp-client-identifier',
        type: 'binary',
        space: 'dhcp4',
    },
    {
        code: 62,
        name: 'nwip-domain-name',
        type: 'string',
        space: 'dhcp4',
    },
    {
        code: 63,
        name: 'nwip-suboptions',
        type: 'binary',
        space: 'dhcp4',
    },
    {
        code: 64,
        name: 'nisplus-domain-name',
        type: 'string',
        space: 'dhcp4',
    },
    {
        code: 65,
        name: 'nisplus-servers',
        type: 'ipv4-address',
        space: 'dhcp4',
    },
    {
        code: 66,
        name: 'tftp-server-name',
        type: 'string',
        space: 'dhcp4',
    },
    {
        code: 67,
        name: 'boot-file-name',
        type: 'string',
        space: 'dhcp4',
    },
    {
        code: 68,
        name: 'mobile-ip-home-agent',
        type: 'ipv4-address',
        space: 'dhcp4',
    },
    {
        code: 69,
        name: 'smtp-server',
        type: 'ipv4-address',
        space: 'dhcp4',
    },
    {
        code: 70,
        name: 'pop-server',
        type: 'ipv4-address',
        space: 'dhcp4',
    },
    {
        code: 71,
        name: 'nntp-server',
        type: 'ipv4-address',
        space: 'dhcp4',
    },
    {
        code: 72,
        name: 'www-server',
        type: 'ipv4-address',
        space: 'dhcp4',
    },
    {
        code: 73,
        name: 'finger-server',
        type: 'ipv4-address',
        space: 'dhcp4',
    },
    {
        code: 74,
        name: 'irc-server',
        type: 'ipv4-address',
        space: 'dhcp4',
    },
    {
        code: 75,
        name: 'streettalk-server',
        type: 'ipv4-address',
        space: 'dhcp4',
    },
    {
        code: 76,
        name: 'streettalk-directory-assistance-server',
        type: 'ipv4-address',
        space: 'dhcp4',
    },
    {
        code: 77,
        name: 'user-class',
        type: 'binary',
        space: 'dhcp4',
    },
    {
        code: 78,
        name: 'slp-directory-agent',
        type: 'record',
        space: 'dhcp4',
        recordTypes: ['bool', 'ipv4-address'],
    },
    {
        code: 79,
        name: 'slp-service-scope',
        type: 'record',
        space: 'dhcp4',
        recordTypes: ['bool', 'string'],
    },
    {
        code: 81,
        name: 'fqdn',
        type: 'record',
        space: 'dhcp4',
        recordTypes: ['uint8', 'uint8', 'uint8', 'fqdn'],
    },
    {
        code: 82,
        name: 'dhcp-agent-options',
        type: 'empty',
        space: 'dhcp4',
        encapsulate: 'dhcp-agent-options-space',
    },
    {
        code: 85,
        name: 'nds-servers',
        type: 'ipv4-address',
        space: 'dhcp4',
    },
    {
        code: 86,
        name: 'nds-tree-name',
        type: 'string',
        space: 'dhcp4',
    },
    {
        code: 87,
        name: 'nds-context',
        type: 'string',
        space: 'dhcp4',
    },
    {
        code: 88,
        name: 'bcms-controller-names',
        type: 'fqdn',
        space: 'dhcp4',
    },
    {
        code: 89,
        name: 'bcms-controller-address',
        type: 'ipv4-address',
        space: 'dhcp4',
    },
    {
        code: 90,
        name: 'authenticate',
        type: 'binary',
        space: 'dhcp4',
    },
    {
        code: 91,
        name: 'client-last-transaction-time',
        type: 'uint32',
        space: 'dhcp4',
    },
    {
        code: 92,
        name: 'associated-ip',
        type: 'ipv4-address',
        space: 'dhcp4',
    },
    {
        code: 93,
        name: 'client-system',
        type: 'uint16',
        space: 'dhcp4',
    },
    {
        code: 94,
        name: 'client-ndi',
        type: 'record',
        space: 'dhcp4',
        recordTypes: ['uint8', 'uint8', 'uint8'],
    },
    {
        code: 97,
        name: 'uuid-guid',
        type: 'record',
        space: 'dhcp4',
        recordTypes: ['uint8', 'binary'],
    },
    {
        code: 98,
        name: 'uap-servers',
        type: 'string',
        space: 'dhcp4',
    },
    {
        code: 99,
        name: 'geoconf-civic',
        type: 'binary',
        space: 'dhcp4',
    },
    {
        code: 100,
        name: 'pcode',
        type: 'string',
        space: 'dhcp4',
    },
    {
        code: 101,
        name: 'tcode',
        type: 'string',
        space: 'dhcp4',
    },
    {
        code: 108,
        name: 'v6-only-preferred',
        type: 'uint32',
        space: 'dhcp4',
    },
    {
        code: 112,
        name: 'netinfo-server-address',
        type: 'ipv4-address',
        space: 'dhcp4',
    },
    {
        code: 113,
        name: 'netinfo-server-tag',
        type: 'string',
        space: 'dhcp4',
    },
    {
        code: 114,
        name: 'v4-captive-portal',
        type: 'string',
        space: 'dhcp4',
    },
    {
        code: 116,
        name: 'auto-config',
        type: 'uint8',
        space: 'dhcp4',
    },
    {
        code: 117,
        name: 'name-service-search',
        type: 'uint16',
        space: 'dhcp4',
    },
    {
        code: 118,
        name: 'subnet-selection',
        type: 'ipv4-address',
        space: 'dhcp4',
    },
    {
        code: 119,
        name: 'domain-search',
        type: 'fqdn',
        space: 'dhcp4',
    },
    {
        code: 124,
        name: 'vivco-suboptions',
        type: 'record',
        space: 'dhcp4',
        recordTypes: ['uint32', 'binary'],
    },
    {
        code: 125,
        name: 'vivso-suboptions',
        type: 'uint32',
        space: 'dhcp4',
    },
    {
        code: 136,
        name: 'pana-agent',
        type: 'ipv4-address',
        space: 'dhcp4',
    },
    {
        code: 137,
        name: 'v4-lost',
        type: 'fqdn',
        space: 'dhcp4',
    },
    {
        code: 138,
        name: 'capwap-ac-v4',
        type: 'ipv4-address',
        space: 'dhcp4',
    },
    {
        code: 141,
        name: 'sip-ua-cs-domains',
        type: 'fqdn',
        space: 'dhcp4',
    },
    {
        code: 146,
        name: 'rdnss-selection',
        type: 'record',
        space: 'dhcp4',
        recordTypes: ['uint8', 'ipv4-address', 'ipv4-address', 'fqdn'],
    },
    {
        code: 151,
        name: 'status-code',
        type: 'record',
        space: 'dhcp4',
        recordTypes: ['uint8', 'string'],
    },
    {
        code: 152,
        name: 'base-time',
        type: 'uint32',
        space: 'dhcp4',
    },
    {
        code: 153,
        name: 'start-time-of-state',
        type: 'uint32',
        space: 'dhcp4',
    },
    {
        code: 154,
        name: 'query-start-time',
        type: 'uint32',
        space: 'dhcp4',
    },
    {
        code: 155,
        name: 'query-end-time',
        type: 'uint32',
        space: 'dhcp4',
    },
    {
        code: 156,
        name: 'dhcp-state',
        type: 'uint8',
        space: 'dhcp4',
    },
    {
        code: 157,
        name: 'data-source',
        type: 'uint8',
        space: 'dhcp4',
    },
    {
        code: 159,
        name: 'v4-portparams',
        type: 'record',
        space: 'dhcp4',
        recordTypes: ['uint8', 'psid'],
    },
    {
        code: 212,
        name: 'option-6rd',
        type: 'record',
        space: 'dhcp4',
        recordTypes: ['uint8', 'uint8', 'ipv6-address', 'ipv4-address'],
    },
    {
        code: 213,
        name: 'v4-access-domain',
        type: 'fqdn',
        space: 'dhcp4',
    },
    {
        code: 1,
        name: 'circuit-id',
        type: 'string',
        space: 'dhcp-agent-options-space',
    },
    {
        code: 2,
        name: 'remote-id',
        type: 'string',
        space: 'dhcp-agent-options-space',
    },
    {
        code: 4,
        name: 'docsis-device-class',
        type: 'uint32',
        space: 'dhcp-agent-options-space',
    },
    {
        code: 5,
        name: 'link-selection',
        type: 'ipv4-address',
        space: 'dhcp-agent-options-space',
    },
    {
        code: 6,
        name: 'subscriber-id',
        type: 'string',
        space: 'dhcp-agent-options-space',
    },
    {
        code: 7,
        name: 'radius',
        type: 'binary',
        space: 'dhcp-agent-options-space',
    },
    {
        code: 8,
        name: 'auth',
        type: 'binary',
        space: 'dhcp-agent-options-space',
    },
    {
        code: 9,
        name: 'vendor-specific-info',
        type: 'binary',
        space: 'dhcp-agent-options-space',
    },
    {
        code: 10,
        name: 'relay-flags',
        type: 'uint8',
        space: 'dhcp-agent-options-space',
    },
    {
        code: 11,
        name: 'server-id-override',
        type: 'ipv4-address',
        space: 'dhcp-agent-options-space',
    },
    {
        code: 12,
        name: 'relay-id',
        type: 'binary',
        space: 'dhcp-agent-options-space',
    },
    {
        code: 13,
        name: 'access-techno-type',
        type: 'uint16',
        space: 'dhcp-agent-options-space',
    },
    {
        code: 14,
        name: 'access-network-name',
        type: 'string',
        space: 'dhcp-agent-options-space',
    },
    {
        code: 15,
        name: 'access-point-name',
        type: 'string',
        space: 'dhcp-agent-options-space',
    },
    {
        code: 16,
        name: 'access-point-bssid',
        type: 'binary',
        space: 'dhcp-agent-options-space',
    },
    {
        code: 17,
        name: 'operator-id',
        type: 'uint32',
        space: 'dhcp-agent-options-space',
    },
    {
        code: 18,
        name: 'operator-realm',
        type: 'string',
        space: 'dhcp-agent-options-space',
    },
    {
        code: 19,
        name: 'relay-port',
        type: 'uint16',
        space: 'dhcp-agent-options-space',
    },
    {
        code: 151,
        name: 'virtual-subnet-select',
        type: 'binary',
        space: 'dhcp-agent-options-space',
    },
    {
        code: 152,
        name: 'virtual-subnet-select-ctrl',
        type: 'empty',
        space: 'dhcp-agent-options-space',
    },
]
