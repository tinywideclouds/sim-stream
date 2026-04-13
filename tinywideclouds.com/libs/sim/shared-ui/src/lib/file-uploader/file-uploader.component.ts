// libs/sim/shared-ui/src/lib/file-uploader/file-uploader.component.ts
import { Component, inject } from '@angular/core';
import { SimulationStateService } from '@tinywideclouds.com/libs/sim/data-access';

export type UploadType = 'actor' | 'power' | 'meter';

@Component({
  selector: 'sim-file-uploader',
  standalone: true,
  templateUrl: './file-uploader.component.html',
  styleUrl: './file-uploader.component.scss'
})
export class FileUploaderComponent {
  private stateService = inject(SimulationStateService);

  onDragOver(event: DragEvent): void {
    event.preventDefault();
  }

  async onDrop(event: DragEvent, type: UploadType): Promise<void> {
    event.preventDefault();
    if (event.dataTransfer?.files && event.dataTransfer.files.length > 0) {
      await this.processFile(event.dataTransfer.files[0], type);
    }
  }

  async onFileSelected(event: Event, type: UploadType): Promise<void> {
    const input = event.target as HTMLInputElement;
    if (input.files && input.files.length > 0) {
      await this.processFile(input.files[0], type);
      input.value = ''; // Reset input so the same file can be re-selected if needed
    }
  }

  private async processFile(file: File, type: UploadType): Promise<void> {
    const text = await file.text();
    if (type === 'actor') {
      this.stateService.loadActorTimelineCsv(text);
    } else if (type === 'power') {
      this.stateService.loadPowerUsageCsv(text);
    } else if (type === 'meter') {
      this.stateService.loadActorMetersCsv(text);
    }
  }
}