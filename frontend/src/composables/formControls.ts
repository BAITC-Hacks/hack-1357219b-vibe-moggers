import type { ObjectDirective } from 'vue'

const listeners = new WeakMap<HTMLTextAreaElement, () => void>()
export const autoGrow: ObjectDirective<HTMLTextAreaElement> = {
  mounted(el) {
    const minimum = el.getBoundingClientRect().height
    const resize = () => {
      el.style.height = 'auto'
      el.style.height = `${Math.max(minimum, el.scrollHeight + 2)}px`
    }
    listeners.set(el, resize)
    el.addEventListener('input', resize)
    resize()
  },
  updated(el) {
    listeners.get(el)?.()
  },
  unmounted(el) {
    const resize = listeners.get(el)
    if (resize) el.removeEventListener('input', resize)
    listeners.delete(el)
  },
}

/** Native constraint metadata, application-owned messages; no browser validation popup. */
export function validateForm(event: Event, errorId: string): string {
  const form = event.target as HTMLFormElement
  const fields = [
    ...form.querySelectorAll<HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement>(
      'input, textarea, select',
    ),
  ]
  const invalid = fields.find((field) => !field.disabled && !field.validity.valid)
  for (const field of fields) {
    field.setAttribute('aria-invalid', String(field === invalid))
    if (field === invalid) field.setAttribute('aria-describedby', errorId)
    else if (field.getAttribute('aria-describedby') === errorId)
      field.removeAttribute('aria-describedby')
  }
  if (!invalid) return ''
  invalid.focus()
  const label = invalid.labels?.[0]?.textContent?.trim() || 'Поле'
  if (invalid.validity.valueMissing) return `Заполните поле «${label}».`
  if (invalid.validity.typeMismatch)
    return `Проверьте формат поля «${label}». Для ссылки укажите полный адрес с https://.`
  if (invalid.validity.tooShort) return `Добавьте подробности в поле «${label}».`
  return `Проверьте поле «${label}»: значение должно соответствовать указанным ограничениям.`
}
