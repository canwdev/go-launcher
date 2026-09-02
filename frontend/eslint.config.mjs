import antfu from '@antfu/eslint-config'

export default antfu(
  {
    vue: true,
    typescript: true,
    ignores: [
      'dist/**',
      'wailsjs/**',
    ],
  },
  {
    rules: {
      'no-alert': 'off',
      // 项目统一使用 kebab-case 自定义事件名（如 @stop-timer / @grid-dragstart）
      'vue/custom-event-name-casing': ['error', 'kebab-case'],
    },
  },
)
