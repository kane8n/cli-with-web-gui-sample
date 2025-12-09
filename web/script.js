class JSONToYAMLConverter {
    constructor() {
        this.initializeElements();
        this.attachEventListeners();
        this.jsonContent = '';
        this.fileName = '';
        this.startHeartbeat();
        this.setupBeforeUnload();
    }

    initializeElements() {
        this.uploadArea = document.getElementById('uploadArea');
        this.fileInput = document.getElementById('fileInput');
        this.jsonContentTextarea = document.getElementById('jsonContent');
        this.convertBtn = document.getElementById('convertBtn');
        this.resultSection = document.getElementById('resultSection');
        this.errorSection = document.getElementById('errorSection');
        this.yamlOutput = document.getElementById('yamlOutput');
        this.errorOutput = document.getElementById('errorOutput');
        this.copyBtn = document.getElementById('copyBtn');
        this.downloadBtn = document.getElementById('downloadBtn');
        this.newConversionBtn = document.getElementById('newConversionBtn');
        this.tryAgainBtn = document.getElementById('tryAgainBtn');
        this.loading = document.getElementById('loading');
    }

    attachEventListeners() {
        // File upload events
        this.uploadArea.addEventListener('click', () => this.fileInput.click());
        this.uploadArea.addEventListener('dragover', this.handleDragOver.bind(this));
        this.uploadArea.addEventListener('dragleave', this.handleDragLeave.bind(this));
        this.uploadArea.addEventListener('drop', this.handleDrop.bind(this));
        this.fileInput.addEventListener('change', this.handleFileSelect.bind(this));

        // Text input events
        this.jsonContentTextarea.addEventListener('input', this.handleTextInput.bind(this));

        // Button events
        this.convertBtn.addEventListener('click', this.handleConvert.bind(this));
        this.copyBtn.addEventListener('click', this.handleCopy.bind(this));
        this.downloadBtn.addEventListener('click', this.handleDownload.bind(this));
        this.newConversionBtn.addEventListener('click', this.handleNewConversion.bind(this));
        this.tryAgainBtn.addEventListener('click', this.handleTryAgain.bind(this));
    }

    handleDragOver(e) {
        e.preventDefault();
        this.uploadArea.classList.add('dragover');
    }

    handleDragLeave(e) {
        e.preventDefault();
        this.uploadArea.classList.remove('dragover');
    }

    handleDrop(e) {
        e.preventDefault();
        this.uploadArea.classList.remove('dragover');

        const files = e.dataTransfer.files;
        if (files.length > 0) {
            this.processFile(files[0]);
        }
    }

    handleFileSelect(e) {
        if (e.target.files.length > 0) {
            this.processFile(e.target.files[0]);
        }
    }

    processFile(file) {
        if (!file.name.toLowerCase().endsWith('.json')) {
            this.showError('Please select a JSON file.');
            return;
        }

        this.fileName = file.name;
        const reader = new FileReader();

        reader.onload = (e) => {
            this.jsonContent = e.target.result;
            this.jsonContentTextarea.value = this.jsonContent;
            this.updateConvertButton();
        };

        reader.onerror = () => {
            this.showError('Error reading file.');
        };

        reader.readAsText(file);
    }

    handleTextInput() {
        this.jsonContent = this.jsonContentTextarea.value;
        this.fileName = 'output.yaml';
        this.updateConvertButton();
    }

    updateConvertButton() {
        this.convertBtn.disabled = !this.jsonContent.trim();
    }

    async handleConvert() {
        if (!this.jsonContent.trim()) {
            this.showError('Please provide JSON content.');
            return;
        }

        this.showLoading();

        try {
            const formData = new FormData();
            formData.append('json_content', this.jsonContent);

            const response = await fetch('/convert', {
                method: 'POST',
                body: formData
            });

            const result = await response.json();

            if (response.ok) {
                this.showResult(result.yaml);
            } else {
                this.showError(result.error || 'Conversion failed');
            }
        } catch (error) {
            this.showError('Network error: ' + error.message);
        } finally {
            this.hideLoading();
        }
    }

    async handleCopy() {
        const yamlContent = this.yamlOutput.textContent;
        try {
            await navigator.clipboard.writeText(yamlContent);
            const originalText = this.copyBtn.textContent;
            this.copyBtn.textContent = 'Copied!';
            setTimeout(() => {
                this.copyBtn.textContent = originalText;
            }, 2000);
        } catch (error) {
            this.showError('Failed to copy to clipboard');
        }
    }

    handleDownload() {
        const yamlContent = this.yamlOutput.textContent;
        const baseName = this.fileName.replace(/\.json$/i, '').replace(/\.yaml$/i, '');
        const filename = `${baseName}.yaml`;

        const blob = new Blob([yamlContent], { type: 'text/yaml' });
        const url = window.URL.createObjectURL(blob);

        const a = document.createElement('a');
        a.style.display = 'none';
        a.href = url;
        a.download = filename;

        document.body.appendChild(a);
        a.click();

        window.URL.revokeObjectURL(url);
        document.body.removeChild(a);
    }

    handleNewConversion() {
        this.resetForm();
        this.hideAllSections();
    }

    handleTryAgain() {
        this.hideAllSections();
    }

    resetForm() {
        this.jsonContent = '';
        this.fileName = '';
        this.jsonContentTextarea.value = '';
        this.fileInput.value = '';
        this.updateConvertButton();
    }

    showLoading() {
        this.loading.style.display = 'flex';
    }

    hideLoading() {
        this.loading.style.display = 'none';
    }

    showResult(yaml) {
        this.yamlOutput.textContent = yaml;
        this.resultSection.style.display = 'block';
        this.errorSection.style.display = 'none';
    }

    showError(message) {
        this.errorOutput.textContent = message;
        this.errorSection.style.display = 'block';
        this.resultSection.style.display = 'none';
    }

    hideAllSections() {
        this.resultSection.style.display = 'none';
        this.errorSection.style.display = 'none';
    }

    startHeartbeat() {
        // Send heartbeat every 2 seconds to keep the server alive
        this.heartbeatInterval = setInterval(async () => {
            try {
                await fetch('/heartbeat', {
                    method: 'POST',
                    cache: 'no-cache'
                });
            } catch (error) {
                // Server might be down, stop heartbeat
                console.log('Heartbeat failed, server may have shut down');
                clearInterval(this.heartbeatInterval);
            }
        }, 2000);

        // Send initial heartbeat
        fetch('/heartbeat', {
            method: 'POST',
            cache: 'no-cache'
        }).catch(() => {});
    }

    setupBeforeUnload() {
        // Stop heartbeat when page is about to be unloaded
        window.addEventListener('beforeunload', () => {
            if (this.heartbeatInterval) {
                clearInterval(this.heartbeatInterval);
            }
        });

        // Also handle when the page visibility changes (tab switch, minimize, etc.)
        document.addEventListener('visibilitychange', () => {
            if (document.hidden) {
                // Page is now hidden, stop heartbeat after a short delay
                setTimeout(() => {
                    if (document.hidden && this.heartbeatInterval) {
                        clearInterval(this.heartbeatInterval);
                    }
                }, 10000); // 10 seconds delay
            } else {
                // Page is visible again, restart heartbeat if needed
                if (!this.heartbeatInterval) {
                    this.startHeartbeat();
                }
            }
        });
    }
}

// Initialize the converter when the DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    new JSONToYAMLConverter();
});
