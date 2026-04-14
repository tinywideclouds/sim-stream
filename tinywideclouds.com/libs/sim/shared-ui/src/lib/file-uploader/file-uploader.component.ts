import { Component, inject, signal } from '@angular/core';
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

  loadingState = signal<Record<UploadType, boolean>>({
    actor: false,
    power: false,
    meter: false
  });

  // NEW: Track the successfully loaded file names
  loadedFiles = signal<Record<UploadType, string | null>>({
    actor: null,
    power: null,
    meter: null
  });

  onDragOver(event: DragEvent): void {
    event.preventDefault();
  }

  async onDrop(event: DragEvent, type: UploadType): Promise<void> {
    event.preventDefault();
    if (this.loadingState()[type]) return; 
    
    if (event.dataTransfer?.files && event.dataTransfer.files.length > 0) {
      await this.processFile(event.dataTransfer.files[0], type);
    }
  }

  async onFileSelected(event: Event, type: UploadType): Promise<void> {
    if (this.loadingState()[type]) return;

    const input = event.target as HTMLInputElement;
    if (input.files && input.files.length > 0) {
      await this.processFile(input.files[0], type);
      input.value = ''; 
    }
  }

  private async processFile(file: File, type: UploadType): Promise<void> {
    this.loadingState.update(state => ({ ...state, [type]: true }));
    
    // Yield to render the spinner
    await new Promise(resolve => setTimeout(resolve, 50));

    try {
      const text = await file.text();
      if (type === 'actor') {
        this.stateService.loadActorTimelineCsv(text);
      } else if (type === 'power') {
        this.stateService.loadPowerUsageCsv(text);
      } else if (type === 'meter') {
        this.stateService.loadActorMetersCsv(text);
      }
      
      // Update the UI to show the file was successfully loaded
      this.loadedFiles.update(state => ({ ...state, [type]: file.name }));
    } catch (error) {
      console.error(`Failed to parse ${type} CSV:`, error);
      alert(`Error parsing ${type} file. Check console for details.`);
    } finally {
      this.loadingState.update(state => ({ ...state, [type]: false }));
    }
  }
}