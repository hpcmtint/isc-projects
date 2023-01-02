/**
 * Attention! Generated Code!
 *
 * Run "rake gen:std_option_defs" to regenerate the option definitions
 * specified in the "codegen/std_dhcpv6_option_def.json", using the
 * template file "std-dhcpv6-option-defs.ts.template", into the
 * "std-dhcpv6-option-defs.ts".
 */

/**
 * Standard DHCPv6 option definitions.
 */
export const stdDhcpv6OptionDefs = [
    {
        code: 94,
        encapsulate: 's46-cont-mape-options',
        name: 's46-cont-mape',
        type: 'empty',
        space: 'dhcp6',
    },
    {
        code: 95,
        encapsulate: 's46-cont-mapt-options',
        name: 's46-cont-mapt',
        type: 'empty',
        space: 'dhcp6',
    },
    {
        code: 96,
        encapsulate: 's46-cont-lw-options',
        name: 's46-cont-lw',
        type: 'empty',
        space: 'dhcp6',
    },
    {
        code: 90,
        encapsulate: '',
        name: 's46-br',
        type: 'ipv6-address',
        space: 's46-cont-mape-options',
    },
    {
        code: 89,
        encapsulate: 's46-rule-options',
        name: 's46-rule',
        type: 'record',
        space: 's46-cont-mape-options',
        recordTypes: ['uint8', 'uint8', 'uint8', 'ipv4-address', 'ipv6-prefix'],
    },
    {
        code: 89,
        encapsulate: 's46-rule-options',
        name: 's46-rule',
        type: 'record',
        space: 's46-cont-mapt-options',
        recordTypes: ['uint8', 'uint8', 'uint8', 'ipv4-address', 'ipv6-prefix'],
    },
    {
        code: 91,
        encapsulate: '',
        name: 's46-dmr',
        type: 'ipv6-prefix',
        space: 's46-cont-mapt-options',
    },
    {
        code: 90,
        encapsulate: '',
        name: 's46-br',
        type: 'ipv6-address',
        space: 's46-cont-lw-options',
    },
    {
        code: 92,
        encapsulate: 's46-v4v6bind-options',
        name: 's46-v4v6bind',
        type: 'record',
        space: 's46-cont-lw-options',
        recordTypes: ['ipv4-address', 'ipv6-prefix'],
    },
    {
        code: 93,
        encapsulate: '',
        name: 's46-portparams',
        type: 'record',
        space: 's46-rule-options',
        recordTypes: ['uint8', 'psid'],
    },
    {
        code: 93,
        encapsulate: '',
        name: 's46-portparams',
        type: 'record',
        space: 's46-v4v6bind-options',
        recordTypes: ['uint8', 'psid'],
    },
]
