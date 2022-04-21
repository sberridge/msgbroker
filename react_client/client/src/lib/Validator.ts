export class ValidationResult {
    public success:boolean;
    public message:string;
    constructor(success:boolean = true,message: string = "") {
        this.success = success;
        this.message = message;
    }
}
type validateFunction = {
    value: any,
    func: (value:any)=>Promise<ValidationResult>,
    field: string,
    object: any
}
export class Validator {
    private fields:object;
    private validationResult:{[key:string]:string};
    public success:boolean;
    private validateFunctions:validateFunction[] = [];
    public modelResults = {};

    constructor(fields:{[key:string]:any}) {
        this.fields = fields;
        this.validationResult = {};
        this.success = true;
    }

    private get(fields:any,parentObject:any,validationObject:any,validationFunc: (value: any) => Promise<ValidationResult>) {
        var field:string = fields.shift();
        if(field.substring(field.length-2) == "[]") {
            var field = field.substring(0,field.length-2);
            var val = (typeof parentObject[field]) == "undefined" ? null : parentObject[field];
            if(fields.length == 0) {
                if(!Array.isArray(val)) {
                    validationObject[field] = "Expected Array";
                    this.success = false;
                } else {
                    validationObject[field] = (field in validationObject) ? validationObject[field] : {};
                    val.forEach((value,index)=>{
                        this.validateFunctions.push({
                            value: value,
                            func: validationFunc,
                            field: index.toString(),
                            object: validationObject[field]
                        })
                    });
                }
            } else {
                validationObject[field] = (field in validationObject) ? validationObject[field] : [];
                if(val != null) {
                    if(!Array.isArray(val)) {
                        validationObject[field] = "Expected Array";
                        this.success = false;
                    } else {
                        val.forEach((item,index)=>{
                            var validObj = {};
                            if(validationObject[field].length > index) {
                                validObj = validationObject[field][index];
                            } else {
                                validationObject[field].push(validObj);
                            }                            
                            this.get(fields.slice(0),item,validObj,validationFunc);
                        });
                    }                    
                }                
            }            
        } else {
            var val = (typeof parentObject[field]) == "undefined" ? null : parentObject[field];         
            if(fields.length == 0) {   
                this.validateFunctions.push({
                    value: val,
                    func: validationFunc,
                    field: field,
                    object: validationObject
                });            
            } else {
                if(!(field in validationObject)) {
                    var nextValidationObject = {};
                    validationObject[field] = nextValidationObject;
                } else {
                    nextValidationObject = validationObject[field];
                }                
                if(val == null) {
                    val = {};
                    parentObject[field] = {};
                }
                this.get(fields,val,nextValidationObject,validationFunc);
            }            
        }
    }

    

    public validateRequired(field:string) {
        var fieldParts = field.split(".");
        this.get(fieldParts,this.fields,this.validationResult,function(value): Promise<ValidationResult> {
            return new Promise((resolve,reject)=>{
                if(value === null) {
                    resolve(new ValidationResult(false,"required"));
                }
                resolve(new ValidationResult());
            });
            
        });
    }
    
    public validateMinLength(field:string,length:number) {
        var fieldParts = field.split(".");
        this.get(fieldParts,this.fields,this.validationResult,function(value): Promise<ValidationResult> {
            return new Promise((resolve,reject)=>{
                if(value !== null && value.length < length) {
                    resolve(new ValidationResult(false,"minimum length: " + length.toString()));
                }
                resolve(new ValidationResult());
            });
            
        });
    }


    public validateDate(field:string) {
        var fieldParts = field.split(".");
        this.get(fieldParts,this.fields,this.validationResult,function(value:string): Promise<ValidationResult> {
            return new Promise((resolve,reject)=>{
                if(value !== null) {
                    if(!/^[0-9]{4}\-[0-9]{2}\-[0-9]{2}$/.test(value) || isNaN((new Date(value)).getTime())) {
                        resolve(new ValidationResult(false,"Invalid date"));
                    }                    
                }
                resolve(new ValidationResult());
            });
            
        });
    }
    
    public validateTime(field:string) {
        var fieldParts = field.split(".");
        this.get(fieldParts,this.fields,this.validationResult,function(value:string): Promise<ValidationResult> {
            return new Promise((resolve,reject)=>{
                if(value !== null) {
                    if(!/^[0-9]{2}\:[0-9]{2}(\:[0-9]{2})?$/.test(value) || isNaN((new Date("1990-01-01 " + value)).getTime())) {
                        resolve(new ValidationResult(false,"Invalid date"));
                    }                    
                }
                resolve(new ValidationResult());
            });
            
        });
    }

    public validateNumeric(field:string) {
        var fieldParts = field.split(".");
        this.get(fieldParts,this.fields,this.validationResult,function(value): Promise<ValidationResult> {
            return new Promise((resolve,reject)=>{
                if(isNaN(value)) {
                    resolve(new ValidationResult(false,"Invalid number"));
                }
                resolve(new ValidationResult());
            });
            
        });
    }

    public validateCustom(field:string,func: (value:any)=>Promise<ValidationResult>) {
        var fieldParts = field.split(".");
        this.get(fieldParts,this.fields,this.validationResult,function(value): Promise<ValidationResult> {
            return new Promise((resolve,reject)=>{
                if(value !== null) {
                    resolve(func(value));
                } else {
                    resolve(new ValidationResult());
                }
            });
        });
    }

    public validateBoolean(field:string) {
        var fieldParts = field.split(".");
        this.get(fieldParts,this.fields,this.validationResult,function(value): Promise<ValidationResult> {
            return new Promise((resolve,reject)=>{
                if(value !== null && typeof value !== "boolean") {
                    resolve(new ValidationResult(false,"boolean required"));
                }
                resolve(new ValidationResult());
            });

        });
    }

    public validateInteger(field:string) {
        var fieldParts = field.split(".");
        this.get(fieldParts,this.fields,this.validationResult,function(value): Promise<ValidationResult> {
            return new Promise((resolve,reject)=>{
                if(value !== null && isNaN(parseInt(value))) {
                    resolve(new ValidationResult(false,"integer required"));
                }
                resolve(new ValidationResult());
            });

        });
    }

    public validateArray(field:string) {
        var fieldParts = field.split(".");
        this.get(fieldParts,this.fields,this.validationResult,function(value): Promise<ValidationResult> {
            return new Promise((resolve,reject)=>{
                if(value !== null && !Array.isArray(value)) {
                    resolve(new ValidationResult(false,"expected array"));
                }
                resolve(new ValidationResult());
            });
            
        });
    }

    public getValidationResult() {
        return this.validationResult;
    }

    public validate():Promise<{[key:string]:string}> {
        return new Promise((resolve,reject)=>{
            var totalFuncs = this.validateFunctions.length;
            var completeFuncs = 0;
            this.validateFunctions.forEach((func)=>{
                func.func(func.value).then((result)=>{
                    if(!result.success) {
                        this.success = false;
                        func.object[func.field] = result.message;
                    }
                    completeFuncs++;
                    if(completeFuncs == totalFuncs) {
                        resolve(this.getValidationResult());
                    }
                });
                
            })
        });
    }

}