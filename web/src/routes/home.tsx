import { PageHeader } from '@/components/page-header'
import { BattlesCard } from '@/features/home/battles-card'
import { StartCard } from '@/features/home/start-card'
import { StatsCard } from '@/features/home/stats-card'

export function Home() {
  return (
    <div className="grid gap-8">
      <PageHeader
        title="Play"
        description="Name your opponent’s song from as few stems as you can."
      />
      <div className="grid items-start gap-5 lg:grid-cols-12">
        <StartCard className="lg:col-span-5" />
        <div className="grid gap-5 lg:col-span-7">
          <StatsCard />
          <BattlesCard />
        </div>
      </div>
    </div>
  )
}
