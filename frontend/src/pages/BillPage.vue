<template>
  <q-page class="q-pa-lg">
    <div class="bill">
      <div v-if="loading" class="row items-center q-gutter-sm q-pa-md">
        <q-spinner size="1.5em" />
        <span>Loading the bill…</span>
      </div>

      <!-- Not rendered at all on a failed load: an empty table would read as
           "nobody is on this bill". -->
      <q-banner v-else-if="error && !data" dense class="bg-negative text-white">
        <template #avatar><q-icon name="error" /></template>
        <div>{{ error.message }}</div>
        <div class="text-caption">{{ error.error_code }}</div>
        <template #action>
          <q-btn flat dense label="Try again" @click="load" />
        </template>
      </q-banner>

      <q-card v-else-if="data" flat bordered>
        <q-card-section>
          <div class="text-h6">{{ data.bill.description }}</div>
          <div class="text-caption text-grey-7">Total</div>
          <div class="text-h5">{{ formatMoney(data.bill.total, data.bill.currency) }}</div>
        </q-card-section>

        <q-separator />

        <q-card-section>
          <q-markup-table flat dense>
            <thead>
              <tr>
                <th class="text-left">Person</th>
                <th class="text-right percentage-column">Percentage</th>
                <th class="text-right amount-column">Amount</th>
                <th class="remove-column"><span class="sr-only">Remove</span></th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="(row, index) in rows" :key="row.share.person_id">
                <td class="text-left">{{ row.share.name }}</td>
                <td class="text-right">
                  <q-input
                    v-model="row.share.percentage"
                    dense
                    outlined
                    hide-bottom-space
                    input-class="text-right"
                    suffix="%"
                    :rules="[isAPercentage]"
                    :aria-label="`${row.share.name}'s percentage`"
                  />
                </td>
                <!-- Recomputed as the Percentage is typed. -->
                <td class="text-right">
                  {{ row.amount === null ? '—' : formatMoney(row.amount, data.bill.currency) }}
                </td>
                <td class="text-center">
                  <!-- Takes the Share off the Bill; the Person stays. -->
                  <q-btn
                    flat
                    dense
                    round
                    icon="close"
                    size="sm"
                    :aria-label="`Remove ${row.share.name} from the bill`"
                    @click="remove(index)"
                  />
                </td>
              </tr>
            </tbody>
            <tfoot>
              <!-- Not "Total": that is the Bill's money, this is its percentages. -->
              <tr :class="runningTotalClass">
                <td class="text-left text-weight-medium">Share set</td>
                <td class="text-right text-weight-medium">{{ runningTotalLabel }}</td>
                <!-- What the column above adds up to. -->
                <td class="text-right text-weight-medium">
                  {{ allocatedTotal === null ? '—' : formatMoney(allocatedTotal, data.bill.currency) }}
                </td>
                <td></td>
              </tr>
            </tfoot>
          </q-markup-table>
        </q-card-section>

        <q-card-section class="q-pt-none">
          <!-- Offers people who already exist, and takes a name that is not among
               them. No fill-input on purpose: leaving the last name in the box
               makes the next one typed append to it. -->
          <q-select
            ref="picker"
            v-model="chosen"
            :options="offered"
            :loading="people.loading.value"
            option-label="name"
            option-value="id"
            use-input
            hide-selected
            input-debounce="0"
            outlined
            dense
            clearable
            label="Add someone to this bill"
            hint="Pick a name, or type a new one and press enter"
            aria-label="Add someone to this bill"
            @filter="filterOffered"
            @update:model-value="onChoose"
            @new-value="onNewName"
          >
            <template #no-option>
              <q-item>
                <q-item-section class="text-grey">
                  Nobody left to add. Type a name to create someone new.
                </q-item-section>
              </q-item>
            </template>
          </q-select>
        </q-card-section>

        <!-- Why saving is blocked, rather than a disabled button to guess at. -->
        <q-card-section v-if="blockedReason" class="q-pt-none">
          <q-banner dense class="bg-warning text-black">
            <template #avatar><q-icon name="warning" /></template>
            {{ blockedReason }}
          </q-banner>
        </q-card-section>

        <!-- Reloading is offered rather than done: refetching over what someone
             typed turns a conflict into data loss. -->
        <q-card-section v-if="conflicted && error" class="q-pt-none">
          <q-banner dense class="bg-warning text-black">
            <template #avatar><q-icon name="sync_problem" /></template>
            <div>{{ error.message }}</div>
            <div class="text-caption">
              Your edits are still here. Reloading replaces them with what is stored.
            </div>
            <template #action>
              <q-btn flat dense label="Reload" @click="load" />
            </template>
          </q-banner>
        </q-card-section>

        <!-- Any other refused save. The edits stay for the same reason. -->
        <q-card-section v-else-if="error && data" class="q-pt-none">
          <q-banner dense class="bg-negative text-white">
            <template #avatar><q-icon name="error" /></template>
            <div>{{ error.message }}</div>
            <div class="text-caption">{{ error.error_code }}</div>
          </q-banner>
        </q-card-section>

        <q-card-actions align="right">
          <q-btn
            color="primary"
            label="Save"
            :disable="!canSave"
            :loading="saving"
            @click="onSave"
          />
        </q-card-actions>
      </q-card>
    </div>
  </q-page>
</template>

<script setup lang="ts">
import { useQuasar, type QSelect } from 'quasar'
import { computed, onMounted, ref } from 'vue'

import { useBill } from '@/composables/useBill'
import { usePeople } from '@/composables/usePeople'
import { isAPercentage, useShareSetDraft } from '@/composables/useShareSetDraft'
import { BILL_ID } from '@/config'
import type { Person } from '@/types/api'
import { formatMoney } from '@/utils/currency'

const quasar = useQuasar()
const { loading, saving, error, conflicted, data, load, save } = useBill(BILL_ID)
const {
  rows,
  allocatedTotal,
  isWhole,
  runningTotalLabel,
  blockedReason,
  add,
  remove,
  notOnTheBill,
  toShareInputs,
} = useShareSetDraft(data)

const people = usePeople()

const chosen = ref<Person | null>(null)
const picker = ref<QSelect | null>(null)
const typed = ref('')

// Derived rather than stored, so the list is right the moment the people arrive.
const offered = computed(() => {
  const candidates = notOnTheBill(people.data.value)
  const needle = typed.value.trim().toLowerCase()
  if (needle === '') return candidates
  return candidates.filter((person) => person.name.toLowerCase().includes(needle))
})

function filterOffered(input: string, update: (fn: () => void) => void) {
  update(() => {
    typed.value = input
  })
}

/**
 * Clearing the bound value is not enough: the typed text stays in the box, and
 * the next name appends to it.
 */
function clearPicker() {
  chosen.value = null
  typed.value = ''
  picker.value?.updateInputValue('')
}

function onChoose(person: Person | null) {
  if (person === null) return
  addToBill(person)
}

/**
 * A name typed rather than picked. Matched against the people who already exist
 * first, because this fires on enter before any highlighted option is taken — so
 * typing a known name would otherwise create a second copy of them.
 */
async function onNewName(name: string) {
  const known = people.findByName(name)
  if (known !== undefined) {
    addToBill(known)
    return
  }

  const result = await people.create(name)
  clearPicker()

  if (!result.ok) {
    quasar.notify({ type: 'negative', message: result.error.message })
    return
  }
  add(result.data)
}

/** Adds someone, or says why they were not added. */
function addToBill(person: Person) {
  const added = add(person)
  clearPicker()

  if (!added) {
    quasar.notify({ type: 'warning', message: `${person.name} is already on this bill.` })
  }
}

// Coloured, so a wrong number is noticed rather than hunted for.
const runningTotalClass = computed(() =>
  isWhole.value && !blockedReason.value ? 'text-positive' : 'text-negative running-total--wrong',
)

const canSave = computed(() => !blockedReason.value && !saving.value)

async function onSave() {
  const result = await save(toShareInputs())

  if (result.ok) {
    quasar.notify({ type: 'positive', message: 'The share set was saved.' })
    return
  }
  // The banner carries the server's message; this says only that nothing saved.
  quasar.notify({
    type: 'negative',
    message: conflicted.value
      ? 'This bill changed elsewhere. Nothing was saved.'
      : 'The share set was not saved.',
  })
}

onMounted(() => {
  // Neither can reject; void says the floating promise is meant.
  void load()
  void people.load()
})
</script>

<style scoped>
.bill {
  max-width: 34rem;
}

.percentage-column {
  width: 11rem;
}

.amount-column {
  width: 7rem;
}

.remove-column {
  width: 3rem;
}

/* Names the column for a screen reader without showing a heading. */
.sr-only {
  position: absolute;
  width: 1px;
  height: 1px;
  overflow: hidden;
  clip: rect(0, 0, 0, 0);
  white-space: nowrap;
}

.running-total--wrong {
  font-weight: 700;
}
</style>
