# Global fee module

The Global fee module is based on [Gaia's implementation](https://github.com/cosmos/gaia). Which is supplied by the great folks at [TGrade](https://github.com/confio/tgrade) 👋, with minor modifications. All credits and big thanks go to the original authors.

More information about Cosmoshub fee system please check [here](https://github.com/cosmos/gaia/blob/v17.3.0/docs/docs/modules/globalfee.md).

## Parameters and governance

The module stores its parameters in its own keeper. Governance updates them with
`/xion.globalfee.v1.MsgUpdateParams`, using the governance module account as
`authority`. The message replaces all three fields in `params`:
`minimum_gas_prices`, `bypass_min_fee_msg_types`, and
`max_total_bypass_min_fee_msg_gas_usage`. Legacy param-change proposals are no
longer supported.

The v32 upgrade migrates the existing values from the legacy `params` store.
Missing legacy fields retain their defaults. Query the current values with
`xiond query globalfee params`.
