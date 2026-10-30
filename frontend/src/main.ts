import { mount } from 'svelte'
import './app.css'
import App from './App.svelte'
import { store } from './lib/store.svelte'
import { tools } from './lib/tools.svelte'

// Lets scripts drive the app while developing.
if (import.meta.env.DEV) Object.assign(window, { store, tools })

const app = mount(App, {
  target: document.getElementById('app')!,
})

export default app
