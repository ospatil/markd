import { Controller } from '@hotwired/stimulus'

export default class extends Controller {
	static targets = ['search']

	focusSearch(e) {
		e.preventDefault()
		this.searchTarget.focus()
		this.searchTarget.select()
	}

	connect() {
		this.handleKeydown = (e) => {
			if ((e.ctrlKey || e.metaKey) && e.key === 'k') {
				e.preventDefault()
				this.searchTarget.focus()
				this.searchTarget.select()
			}
			if (e.key === 'Escape' && document.activeElement === this.searchTarget) {
				this.searchTarget.blur()
			}
		}
		document.addEventListener('keydown', this.handleKeydown)
	}

	disconnect() {
		document.removeEventListener('keydown', this.handleKeydown)
	}
}
