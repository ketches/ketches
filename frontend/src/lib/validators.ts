import { defineRule } from 'vee-validate'

defineRule('slug', (value: string) => {
    if (!value || !value.length) {
        return '此字段为必填项'
    }
    if (value.length < 3 || value.length > 32) {
        return '长度必须在 3 到 32 个字符之间'
    }
    if (!/^[a-z]/.test(value)) {
        return '必须以小写字母开头'
    }
    if (/-$/.test(value)) {
        return '不能以短横线结尾'
    }
    if (!/^[a-z0-9-]+$/.test(value)) {
        return '只能包含小写字母、数字和短横线'
    }
    return true
})
