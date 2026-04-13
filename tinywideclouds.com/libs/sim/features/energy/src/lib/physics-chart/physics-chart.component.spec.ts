import { ComponentFixture, TestBed } from '@angular/core/testing';
import { PhysicsChartComponent } from './physics-chart.component';

describe('PhysicsChartComponent', () => {
  let component: PhysicsChartComponent;
  let fixture: ComponentFixture<PhysicsChartComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [PhysicsChartComponent],
    }).compileComponents();

    fixture = TestBed.createComponent(PhysicsChartComponent);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});