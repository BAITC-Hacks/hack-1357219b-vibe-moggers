<script setup lang="ts">
import { ref, onBeforeUnmount } from 'vue'
const dialog = ref<HTMLDialogElement | null>(null)
let resolve: ((leave: boolean) => void) | null = null
function finish(leave: boolean) {
  dialog.value?.close()
  resolve?.(leave)
  resolve = null
}
function ask(): Promise<boolean> {
  resolve?.(false)
  dialog.value?.showModal()
  return new Promise((done) => {
    resolve = done
  })
}
onBeforeUnmount(() => resolve?.(false))
defineExpose({ ask })
</script>
<template>
  <dialog
    ref="dialog"
    class="leave-dialog"
    aria-labelledby="leave-title"
    aria-describedby="leave-description"
    @cancel.prevent="finish(false)"
  >
    <h2 id="leave-title">Выйти без сохранения?</h2>
    <p id="leave-description">
      Последние изменения в карточке будут потеряны. Останьтесь на странице, чтобы сохранить их.
    </p>
    <div class="editor-actions">
      <button type="button" class="button primary" autofocus @click="finish(false)">Остаться</button
      ><button type="button" class="button secondary" @click="finish(true)">
        Выйти без сохранения
      </button>
    </div>
  </dialog>
</template>
