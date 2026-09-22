import { Notify, Quasar } from 'quasar'
import { createApp } from 'vue'

import App from '@/App.vue'

import '@quasar/extras/material-icons/material-icons.css'
import 'quasar/src/css/index.sass'

createApp(App).use(Quasar, { plugins: { Notify } }).mount('#app')
