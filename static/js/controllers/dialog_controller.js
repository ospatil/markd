import { Controller } from "@hotwired/stimulus"

export default class extends Controller {
  open() {
    this.element.showModal()
  }

  close() {
    this.element.close()
    this.element.querySelector("form")?.reset()
    const errors = this.element.querySelector("#form-errors")
    if (errors) errors.innerHTML = ""
  }

  backdropClose(e) {
    if (e.target === this.element) this.close()
  }
}
