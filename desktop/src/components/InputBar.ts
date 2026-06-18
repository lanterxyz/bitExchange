export class InputBar {
  private el: HTMLElement;
  private onSendText: (text: string, encrypted: boolean) => void;
  private onSendFile: (file: File, encrypted: boolean) => void;
  private encrypted = false;

  constructor(
    el: HTMLElement,
    onSendText: (text: string, encrypted: boolean) => void,
    onSendFile: (file: File, encrypted: boolean) => void
  ) {
    this.el = el;
    this.onSendText = onSendText;
    this.onSendFile = onSendFile;
    this.render();
  }

  private render() {
    this.el.className = 'input-bar';
    this.el.innerHTML = `
      <input class="input-field" type="text" placeholder="输入消息，或拖入/粘贴文件..." />
      <button class="btn" id="file-btn">📎 文件</button>
      <button class="btn btn-toggle" id="encrypt-btn">🔒 加密</button>
      <button class="btn btn-primary" id="send-btn">发送 →</button>
      <input type="file" id="file-input" style="display:none" multiple />
    `;

    const input = this.el.querySelector('.input-field') as HTMLInputElement;
    const fileInput = this.el.querySelector('#file-input') as HTMLInputElement;
    const encryptBtn = this.el.querySelector('#encrypt-btn') as HTMLButtonElement;

    this.el.querySelector('#send-btn')!.addEventListener('click', () => {
      if (input.value.trim()) {
        this.onSendText(input.value, this.encrypted);
        input.value = '';
      }
    });

    this.el.querySelector('#file-btn')!.addEventListener('click', () => fileInput.click());

    fileInput.addEventListener('change', () => {
      if (fileInput.files && fileInput.files.length) {
        for (const f of Array.from(fileInput.files)) {
          this.onSendFile(f, this.encrypted);
        }
      }
      fileInput.value = '';
    });

    input.addEventListener('keydown', (e) => {
      if (e.key === 'Enter') {
        this.el.querySelector('#send-btn')!.dispatchEvent(new Event('click'));
      }
    });

    this.el.addEventListener('dragover', (e) => e.preventDefault());
    this.el.addEventListener('drop', (e) => {
      e.preventDefault();
      if (e.dataTransfer?.files) {
        for (const f of Array.from(e.dataTransfer.files)) {
          this.onSendFile(f, this.encrypted);
        }
      }
    });

    encryptBtn.addEventListener('click', () => {
      this.encrypted = !this.encrypted;
      encryptBtn.classList.toggle('active', this.encrypted);
    });
  }
}
