// libs/sim/shared-ui/src/lib/file-uploader/file-uploader.component.spec.ts
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { FileUploaderComponent } from './file-uploader.component';
import { SimulationStateService } from '@tinywideclouds.com/libs/sim/data-access';

describe('FileUploaderComponent', () => {
  let component: FileUploaderComponent;
  let fixture: ComponentFixture<FileUploaderComponent>;
  
  // Create a framework-agnostic mock object to spy on
  const mockStateService = {
    loadActorTimelineCsv: (text: string) => {},
    loadPowerUsageCsv: (text: string) => {},
    loadActorMetersCsv: (text: string) => {}
  };

  beforeEach(async () => {
    spyOn(mockStateService, 'loadActorTimelineCsv');
    spyOn(mockStateService, 'loadPowerUsageCsv');
    spyOn(mockStateService, 'loadActorMetersCsv');

    await TestBed.configureTestingModule({
      imports: [FileUploaderComponent],
      providers: [
        { provide: SimulationStateService, useValue: mockStateService }
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(FileUploaderComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('should process actor timeline csv drops', async () => {
    const mockFile = new File(['mock content'], 'actor.csv', { type: 'text/csv' });
    await component['processFile'](mockFile, 'actor');
    expect(mockStateService.loadActorTimelineCsv).toHaveBeenCalledWith('mock content');
  });

  it('should process power usage csv drops', async () => {
    const mockFile = new File(['mock content'], 'power.csv', { type: 'text/csv' });
    await component['processFile'](mockFile, 'power');
    expect(mockStateService.loadPowerUsageCsv).toHaveBeenCalledWith('mock content');
  });

  it('should process actor meters csv drops', async () => {
    const mockFile = new File(['mock content'], 'meter.csv', { type: 'text/csv' });
    await component['processFile'](mockFile, 'meter');
    expect(mockStateService.loadActorMetersCsv).toHaveBeenCalledWith('mock content');
  });
});