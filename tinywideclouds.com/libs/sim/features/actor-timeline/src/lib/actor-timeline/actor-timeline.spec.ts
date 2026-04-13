import { ComponentFixture, TestBed } from '@angular/core/testing';
import { ActorTimelineComponent } from './actor-timeline';

describe('ActorTimelineComponent', () => {
  let component: ActorTimelineComponent;
  let fixture: ComponentFixture<ActorTimelineComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [ActorTimelineComponent],
    }).compileComponents();

    fixture = TestBed.createComponent(ActorTimelineComponent);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});