import { Application } from "@hotwired/stimulus"
import ShortcutsController from "./controllers/shortcuts_controller.js"
import DialogController from "./controllers/dialog_controller.js"

const app = Application.start()
app.register("shortcuts", ShortcutsController)
app.register("dialog", DialogController)
