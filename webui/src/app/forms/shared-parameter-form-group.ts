import {
    AbstractControl,
    AbstractControlOptions,
    AsyncValidatorFn,
    FormArray,
    FormControl,
    FormGroup,
    ValidatorFn,
} from '@angular/forms'
import { LinkedFormGroup } from './linked-form-group'

/**
 * Type of the form group holding a shared parameter.
 */
interface SharedParameterForm {
    /**
     * A control indicating if the parameter is unlocked for editing
     * different values for different servers.
     */
    unlocked: FormControl

    /**
     * Controls for the parameter values for different servers.
     */
    values: FormArray
}

/**
 * A shared parameter descriptor in the form group.
 *
 * It provides the metadata for each shared parameter describing its
 * type, selectable values, min and max value, fractional digits and
 * a validation error text.
 *
 * @typeParam type of the parameter values.
 */
interface EditableParameterSpec<T> {
    type: string
    values?: T[]
    min?: number
    max?: number
    fractionDigits?: number
    invalidText?: string
}

/**
 * Extends the FormGroup with custom data of selected type.
 *
 * The FormGroup class is not well suited for the forms with changing
 * set of controls that can't be determined upfront. In that case, it
 * is useful to hold additional information with the form group that,
 * for example, indicates the type of the data, an identifier of the
 * input box, etc. This class derives from the FormGroup (behaves like
 * the FormGroup) and holds such additional custom information.
 *
 * Even though the FormGroup is marked final, deriving from it should be
 * safe in this particular case. The derived class does not call any
 * protected methods and is independent of the base class's API.
 */
export class SharedParameterFormGroup<
    TDataType,
    TControl extends { [K in keyof TControl]: AbstractControl<any> } = any
> extends LinkedFormGroup<EditableParameterSpec<TDataType>, SharedParameterForm> {
    /**
     * Constructor.
     *
     * @param data custom data.
     * @param controls form controls belonging to the form group.
     * @param validatorOrOpts validators or control options.
     * @param asyncValidator asynchronous validators.
     */
    constructor(
        data: EditableParameterSpec<TDataType>,
        controls: FormControl<TDataType>[],
        validatorOrOpts?: ValidatorFn | AbstractControlOptions | ValidatorFn[],
        asyncValidator?: AsyncValidatorFn | AsyncValidatorFn[]
    ) {
        let fgControls = {
            unlocked: new FormControl(!!(controls?.length > 1 && controls.some((c) => c.value != controls[0].value))),
            values: new FormArray(controls),
        }
        super(data, fgControls, validatorOrOpts, asyncValidator)
        this.data = data
    }
}
